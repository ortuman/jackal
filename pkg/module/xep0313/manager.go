// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xep0313

import (
	"context"
	"errors"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/ortuman/jackal/pkg/hook"
	archivemodel "github.com/ortuman/jackal/pkg/model/archive"
	"github.com/ortuman/jackal/pkg/module/xep0004"
	"github.com/ortuman/jackal/pkg/module/xep0059"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/storage/repository"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	dateTimeFormat = "2006-01-02T15:04:05Z"

	defaultPageSize = 50
	maxPageSize     = 250
)

// Manager represents an archive manager.
type Manager struct {
	router       router.Router
	hk           *hook.Hooks
	rep          repository.Repository
	logger       kitlog.Logger
	maxQueueSize int
}

// NewManager returns a new archive manager instance.
func NewManager(
	router router.Router,
	hk *hook.Hooks,
	rep repository.Repository,
	maxQueueSize int,
	logger kitlog.Logger,
) *Manager {
	return &Manager{
		router:       router,
		hk:           hk,
		rep:          rep,
		maxQueueSize: maxQueueSize,
		logger:       logger,
	}
}

// ProcessIQ processes a MAM IQ.
func (m *Manager) ProcessIQ(ctx context.Context, iq *stravaganza.IQ, onArchiveRequestedFn func(archiveID string) error) error {
	switch {
	case iq.IsGet() && iq.ChildNamespace("metadata", mamNamespace) != nil:
		return m.queryMetadata(ctx, iq)

	case iq.IsGet() && iq.ChildNamespace("query", mamNamespace) != nil:
		return m.formFields(ctx, iq)

	case iq.IsSet() && iq.ChildNamespace("query", mamNamespace) != nil:
		if err := m.queryArchive(ctx, iq); err != nil {
			return err
		}
		if onArchiveRequestedFn != nil {
			archiveID := iq.ToJID().ToBareJID().String()
			return onArchiveRequestedFn(archiveID)
		}
	}
	return nil
}

// ArchiveMessage archives a message.
func (m *Manager) ArchiveMessage(ctx context.Context, message *stravaganza.Message, archiveID, id string) error {
	archiveMsg := &archivemodel.Message{
		ArchiveId: archiveID,
		Id:        id,
		FromJid:   message.FromJID().String(),
		ToJid:     message.ToJID().String(),
		Message:   message.Proto(),
		Stamp:     timestamppb.Now(),
	}
	err := m.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		err := tx.InsertArchiveMessage(ctx, archiveMsg)
		if err != nil {
			return err
		}
		return tx.DeleteArchiveOldestMessages(ctx, archiveID, m.maxQueueSize)
	})
	if err != nil {
		return err
	}
	return m.runHook(ctx, hook.ArchiveMessageArchived, &hook.MamInfo{
		ArchiveID: archiveID,
		Message:   archiveMsg,
	})
}

// DeleteArchive deletes an archive.
func (m *Manager) DeleteArchive(ctx context.Context, archiveID string) error {
	return m.rep.DeleteArchive(ctx, archiveID)
}

func (m *Manager) formFields(ctx context.Context, iq *stravaganza.IQ) error {
	form := xep0004.DataForm{
		Type: xep0004.Form,
	}

	form.Fields = append(form.Fields, xep0004.Field{
		Type:   xep0004.Hidden,
		Var:    xep0004.FormType,
		Values: []string{mamNamespace},
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Type: xep0004.JidSingle,
		Var:  "with",
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Type: xep0004.TextSingle,
		Var:  "start",
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Type: xep0004.TextSingle,
		Var:  "end",
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Type: xep0004.TextSingle,
		Var:  "before-id",
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Type: xep0004.TextSingle,
		Var:  "after-id",
	})
	form.Fields = append(form.Fields, xep0004.Field{
		Type: xep0004.ListMulti,
		Var:  "ids",
		Validate: &xep0004.Validate{
			DataType:  xep0004.StringDataType,
			Validator: &xep0004.OpenValidator{},
		},
	})

	qChild := stravaganza.NewBuilder("query").
		WithAttribute(stravaganza.Namespace, mamNamespace).
		WithChild(form.Element()).
		Build()

	_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, qChild))

	level.Info(m.logger).Log("msg", "requested form fields")

	return nil
}

func (m *Manager) queryMetadata(ctx context.Context, iq *stravaganza.IQ) error {
	archiveID := iq.FromJID().ToBareJID().String()

	metadata, err := m.rep.FetchArchiveMetadata(ctx, archiveID)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// send reply
	metadataBuilder := stravaganza.NewBuilder("metadata").WithAttribute(stravaganza.Namespace, mamNamespace)

	startBuilder := stravaganza.NewBuilder("start")
	if metadata != nil {
		startBuilder.WithAttribute("id", metadata.StartId)
		startBuilder.WithAttribute("timestamp", metadata.StartTimestamp)
	}
	endBuilder := stravaganza.NewBuilder("end")
	if metadata != nil {
		endBuilder.WithAttribute("id", metadata.EndId)
		endBuilder.WithAttribute("timestamp", metadata.EndTimestamp)
	}

	metadataBuilder.WithChildren(startBuilder.Build(), endBuilder.Build())

	resIQ := xmpputil.MakeResultIQ(iq, metadataBuilder.Build())
	_, _ = m.router.Route(ctx, resIQ)

	level.Info(m.logger).Log("msg", "requested archive metadata", "archive_id", archiveID)

	return nil
}

func (m *Manager) queryArchive(ctx context.Context, iq *stravaganza.IQ) error {
	qChild := iq.ChildNamespace("query", mamNamespace)

	// filter archive result
	filters := &archivemodel.Filters{}
	if x := qChild.ChildNamespace("x", xep0004.FormNamespace); x != nil {
		form, err := xep0004.NewFormFromElement(x)
		if err != nil {
			return err
		}
		filters, err = formToFilters(form)
		if err != nil {
			return err
		}
	}
	archiveID := iq.FromJID().ToBareJID().String()

	messages, err := m.rep.FetchArchiveMessages(ctx, filters, archiveID)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// run archive queried event
	if err := m.runHook(ctx, hook.ArchiveMessageQueried, &hook.MamInfo{
		ArchiveID: archiveID,
		Filters:   filters,
	}); err != nil {
		return err
	}

	// return not found error if any requested id cannot be found
	switch {
	case len(filters.Ids) > 0 && (len(messages) != len(filters.Ids)):
		fallthrough

	case (len(filters.AfterId) > 0 || len(filters.BeforeId) > 0) && len(messages) == 0:
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.ItemNotFound))
		return nil
	}

	// apply RSM paging
	var req *xep0059.Request
	var res *xep0059.Result

	if set := qChild.ChildNamespace("set", xep0059.RSMNamespace); set != nil {
		req, err = xep0059.NewRequestFromElement(set)
		if err != nil {
			_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
			return err
		}
		if req.Max > maxPageSize {
			req.Max = maxPageSize
		}
	} else {
		req = &xep0059.Request{Max: defaultPageSize}
	}
	messages, res, err = xep0059.GetResultSetPage(messages, req, func(m *archivemodel.Message) string {
		return m.Id
	})
	if err != nil {
		if errors.Is(err, xep0059.ErrPageNotFound) {
			_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.ItemNotFound))
			return nil
		}
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}

	// flip result page
	if qChild.Child("flip-page") != nil {
		messages = lo.Reverse(messages)

		lastID := res.Last
		res.Last = res.First
		res.First = lastID
	}

	// route archive messages
	for _, msg := range messages {
		msgStanza, _ := stravaganza.NewBuilderFromProto(msg.Message).
			BuildStanza()
		stamp := msg.Stamp.AsTime()

		resultElem := stravaganza.NewBuilder("result").
			WithAttribute(stravaganza.Namespace, mamNamespace).
			WithAttribute("queryid", qChild.Attribute("queryid")).
			WithAttribute(stravaganza.ID, uuid.New().String()).
			WithChild(xmpputil.MakeForwardedStanza(msgStanza, &stamp)).
			Build()

		archiveMsg, _ := stravaganza.NewMessageBuilder().
			WithAttribute(stravaganza.From, iq.ToJID().String()).
			WithAttribute(stravaganza.To, iq.FromJID().String()).
			WithAttribute(stravaganza.ID, uuid.New().String()).
			WithChild(resultElem).
			BuildMessage()

		_, _ = m.router.Route(ctx, archiveMsg)
	}

	finB := stravaganza.NewBuilder("fin").
		WithChild(res.Element()).
		WithAttribute(stravaganza.Namespace, mamNamespace)
	if res.Complete {
		finB.WithAttribute("complete", "true")
	}
	_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, finB.Build()))

	level.Info(m.logger).Log("msg", "archive messages requested", "archive_id", archiveID, "count", len(messages), "complete", res.Complete)

	return nil
}

func (m *Manager) runHook(ctx context.Context, hookName string, inf *hook.MamInfo) error {
	_, err := m.hk.Run(hookName, &hook.ExecutionContext{
		Info:    inf,
		Sender:  m,
		Context: ctx,
	})
	return err
}

func formToFilters(fm *xep0004.DataForm) (*archivemodel.Filters, error) {
	var retVal archivemodel.Filters

	fmType := fm.Fields.ValueForFieldOfType(xep0004.FormType, xep0004.Hidden)
	if fm.Type != xep0004.Submit || fmType != mamNamespace {
		return nil, errors.New("unexpected form type value")
	}
	if start := fm.Fields.ValueForField("start"); len(start) > 0 {
		startTm, err := time.Parse(dateTimeFormat, start)
		if err != nil {
			return nil, err
		}
		retVal.Start = timestamppb.New(startTm)
	}
	if end := fm.Fields.ValueForField("end"); len(end) > 0 {
		endTm, err := time.Parse(dateTimeFormat, end)
		if err != nil {
			return nil, err
		}
		retVal.End = timestamppb.New(endTm)
	}
	if with := fm.Fields.ValueForField("with"); len(with) > 0 {
		retVal.With = with
	}
	if beforeID := fm.Fields.ValueForField("before-id"); len(beforeID) > 0 {
		retVal.BeforeId = beforeID
	}
	if afterID := fm.Fields.ValueForField("after-id"); len(afterID) > 0 {
		retVal.AfterId = afterID
	}
	if ids := fm.Fields.ValuesForField("ids"); len(ids) > 0 {
		retVal.Ids = ids
	}
	return &retVal, nil
}

// IsMessageArchievable returns true if the message is archievable.
func IsMessageArchievable(msg *stravaganza.Message) bool {
	return (msg.IsNormal() || msg.IsChat()) && msg.IsMessageWithBody()
}
