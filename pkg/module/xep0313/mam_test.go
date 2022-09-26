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
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/pkg/hook"
	archivemodel "github.com/ortuman/jackal/pkg/model/archive"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
	"github.com/ortuman/jackal/pkg/module/xep0004"
	"github.com/ortuman/jackal/pkg/module/xep0059"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/ortuman/jackal/pkg/storage/repository"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMam_FormFields(t *testing.T) {
	// given
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	mam := &Mam{
		mng:    NewManager(routerMock, nil, nil, 100, kitlog.NewNopLogger()),
		router: routerMock,
		logger: kitlog.NewNopLogger(),
	}

	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "form1").
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/chamber").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, mamNamespace).
				Build(),
		).
		BuildIQ()

	// when
	_ = mam.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 1)
	require.Equal(t, "iq", respStanzas[0].Name())
	require.Equal(t, stravaganza.ResultType, respStanzas[0].Type())

	qChild := respStanzas[0].ChildNamespace("query", mamNamespace)
	require.NotNil(t, qChild)

	x := qChild.ChildNamespace("x", xep0004.FormNamespace)
	require.NotNil(t, x)

	form, _ := xep0004.NewFormFromElement(x)
	require.NotNil(t, form)

	require.Len(t, form.Fields, 7)
}

func TestMam_Metadata(t *testing.T) {
	// given
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	repMock := &repositoryMock{}
	repMock.FetchArchiveMetadataFunc = func(ctx context.Context, archiveID string) (*archivemodel.Metadata, error) {
		return &archivemodel.Metadata{
			StartId:        "s0",
			StartTimestamp: "2008-08-22T21:09:04Z",
			EndId:          "e0",
			EndTimestamp:   "2020-04-20T14:34:21Z",
		}, nil
	}
	mam := &Mam{
		mng:    NewManager(routerMock, hook.NewHooks(), repMock, 100, kitlog.NewNopLogger()),
		hk:     hook.NewHooks(),
		router: routerMock,
		logger: kitlog.NewNopLogger(),
	}

	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "form1").
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/chamber").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("metadata").
				WithAttribute(stravaganza.Namespace, mamNamespace).
				Build(),
		).
		BuildIQ()

	// when
	_ = mam.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 1)
	require.Equal(t, "iq", respStanzas[0].Name())
	require.Equal(t, stravaganza.ResultType, respStanzas[0].Type())

	metadata := respStanzas[0].ChildNamespace("metadata", mamNamespace)
	require.NotNil(t, metadata)

	start := metadata.Child("start")
	require.NotNil(t, start)
	require.Equal(t, "s0", start.Attribute("id"))
	require.Equal(t, "2008-08-22T21:09:04Z", start.Attribute("timestamp"))

	end := metadata.Child("end")
	require.NotNil(t, start)
	require.Equal(t, "e0", end.Attribute("id"))
	require.Equal(t, "2020-04-20T14:34:21Z", end.Attribute("timestamp"))
}

func TestMam_ArchiveMessage(t *testing.T) {
	// given
	var archivedMessages []*archivemodel.Message

	txMock := &txMock{}
	txMock.DeleteArchiveOldestMessagesFunc = func(ctx context.Context, archiveID string, maxElements int) error {
		return nil
	}
	txMock.InsertArchiveMessageFunc = func(ctx context.Context, message *archivemodel.Message) error {
		archivedMessages = append(archivedMessages, message)
		return nil
	}

	repMock := &repositoryMock{}
	repMock.InTransactionFunc = func(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
		return f(ctx, txMock)
	}

	hosts := &hostsMock{}
	hosts.IsLocalHostFunc = func(h string) bool { return h == "jackal.im" }

	hk := hook.NewHooks()
	mam := &Mam{
		mng:    NewManager(nil, hk, repMock, 100, kitlog.NewNopLogger()),
		hk:     hk,
		hosts:  hosts,
		logger: kitlog.NewNopLogger(),
	}
	_ = mam.Start(context.Background())
	t.Cleanup(func() {
		_ = mam.Stop(context.Background())
	})

	msg := testMessageStanzaWithParameters("b0", "ortuman@jackal.im/chamber", "noelia@jackal.im/yard")

	// when
	execCtx := &hook.ExecutionContext{
		Info: &hook.C2SStreamInfo{
			Element: msg,
		},
		Context: context.Background(),
	}
	_, err := hk.Run(hook.C2SStreamMessageReceived, execCtx)
	require.NoError(t, err)

	_, err = hk.Run(hook.C2SStreamMessageRouted, execCtx)
	require.NoError(t, err)

	// then
	require.NoError(t, err)
	require.Len(t, archivedMessages, 2)

	require.Equal(t, "ortuman@jackal.im", archivedMessages[0].ArchiveId)
	require.Equal(t, "noelia@jackal.im", archivedMessages[1].ArchiveId)

	require.Len(t, txMock.DeleteArchiveOldestMessagesCalls(), 2)
	require.Len(t, txMock.InsertArchiveMessageCalls(), 2)

	require.True(t, len(ExtractSentArchiveID(execCtx.Context)) > 0)
	require.True(t, len(ExtractReceivedArchiveID(execCtx.Context)) > 0)
}

func TestMam_SendArchiveMessages(t *testing.T) {
	// given
	archiveMessages := []*archivemodel.Message{
		{
			ArchiveId: "ortuman",
			Stamp:     timestamppb.New(time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)),
			FromJid:   "ortuman@jackal.im/chamber",
			ToJid:     "noelia@jackal.im/yard",
			Message: testMessageStanzaWithParameters(
				"b0",
				"ortuman@jackal.im/chamber",
				"noelia@jackal.im/yard",
			).Proto(),
		},
		{
			ArchiveId: "ortuman",
			Stamp:     timestamppb.New(time.Date(2022, 01, 01, 01, 00, 00, 00, time.UTC)),
			FromJid:   "noelia@jackal.im/yard",
			ToJid:     "ortuman@jackal.im/chamber",
			Message: testMessageStanzaWithParameters(
				"b1",
				"noelia@jackal.im/yard",
				"ortuman@jackal.im/chamber",
			).Proto(),
		},
		{
			ArchiveId: "ortuman",
			Stamp:     timestamppb.New(time.Date(2022, 01, 01, 02, 00, 00, 00, time.UTC)),
			FromJid:   "ortuman@jackal.im/chamber",
			ToJid:     "noelia@jackal.im/yard",
			Message: testMessageStanzaWithParameters(
				"b2",
				"ortuman@jackal.im/chamber",
				"noelia@jackal.im/yard",
			).Proto(),
		},
	}

	c2sInf := c2smodel.NewInfoMap()

	stmMock := &c2sStreamMock{}
	stmMock.SetInfoValueFunc = func(ctx context.Context, k string, val interface{}) error {
		bVal, ok := val.(bool)
		if !ok {
			return errors.New("unexpected value type")
		}
		c2sInf.SetBool(k, bVal)
		return nil
	}

	c2sRouterMock := &c2sRouterMock{}
	c2sRouterMock.LocalStreamFunc = func(username string, resource string) (stream.C2S, error) {
		return stmMock, nil
	}

	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	routerMock.C2SFunc = func() router.C2SRouter {
		return c2sRouterMock
	}

	repMock := &repositoryMock{}
	repMock.FetchArchiveMessagesFunc = func(ctx context.Context, f *archivemodel.Filters, archiveID string) ([]*archivemodel.Message, error) {
		return archiveMessages, nil
	}

	mam := &Mam{
		mng:    NewManager(routerMock, hook.NewHooks(), repMock, 100, kitlog.NewNopLogger()),
		hk:     hook.NewHooks(),
		router: routerMock,
		logger: kitlog.NewNopLogger(),
	}

	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "ortuman1").
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/chamber").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, mamNamespace).
				Build(),
		).
		BuildIQ()

	// when
	_ = mam.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 4) // 3 messages + result iq

	require.Equal(t, stravaganza.MessageName, respStanzas[0].Name())
	require.Equal(t, stravaganza.MessageName, respStanzas[1].Name())
	require.Equal(t, stravaganza.MessageName, respStanzas[2].Name())
	require.Equal(t, stravaganza.IQName, respStanzas[3].Name())

	iqRes := respStanzas[3]
	require.Equal(t, stravaganza.ResultType, iqRes.Type())

	finElem := iqRes.ChildNamespace("fin", mamNamespace)
	require.NotNil(t, finElem)

	rsmRes := finElem.ChildNamespace("set", xep0059.RSMNamespace)
	require.NotNil(t, rsmRes)

	count := rsmRes.Child("count")
	require.NotNil(t, count)
	require.Equal(t, "3", count.Text())

	require.Len(t, stmMock.SetInfoValueCalls(), 1)
	require.True(t, IsArchiveRequested(c2sInf))
}

func TestMam_Forbidden(t *testing.T) {
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	repMock := &repositoryMock{}

	mam := &Mam{
		mng:    NewManager(routerMock, hook.NewHooks(), repMock, 100, kitlog.NewNopLogger()),
		router: routerMock,
		logger: kitlog.NewNopLogger(),
	}

	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "ortuman1").
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithAttribute(stravaganza.From, "noelia@jackal.im/chamber").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, mamNamespace).
				Build(),
		).
		BuildIQ()

	// when
	_ = mam.ProcessIQ(context.Background(), iq)

	require.Len(t, respStanzas, 1)
	require.Equal(t, stravaganza.ErrorType, respStanzas[0].Attribute(stravaganza.Type))
}

func TestMam_DeleteArchive(t *testing.T) {
	// given
	var deletedArchiveID string

	repMock := &repositoryMock{}
	repMock.DeleteArchiveFunc = func(ctx context.Context, archiveID string) error {
		deletedArchiveID = archiveID
		return nil
	}

	hosts := &hostsMock{}
	hosts.IsLocalHostFunc = func(h string) bool { return h == "jackal.im" }

	hk := hook.NewHooks()
	mam := &Mam{
		mng:    NewManager(nil, hk, repMock, 100, kitlog.NewNopLogger()),
		hk:     hk,
		hosts:  hosts,
		logger: kitlog.NewNopLogger(),
	}
	_ = mam.Start(context.Background())
	t.Cleanup(func() {
		_ = mam.Stop(context.Background())
	})

	// when
	_, err := hk.Run(hook.UserDeleted, &hook.ExecutionContext{
		Info: &hook.UserInfo{
			Username: "ortuman",
		},
		Context: context.Background(),
	},
	)

	// then
	require.NoError(t, err)
	require.Len(t, repMock.DeleteArchiveCalls(), 1)

	require.Equal(t, "ortuman", deletedArchiveID)
}

func testMessageStanzaWithParameters(body, from, to string) *stravaganza.Message {
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", from)
	b.WithAttribute("to", to)
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText(body).
			Build(),
	)
	msg, _ := b.BuildMessage()
	return msg
}
