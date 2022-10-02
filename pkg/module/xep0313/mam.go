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

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/storage/repository"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

const (
	// ModuleName represents mam module name.
	ModuleName = "mam"

	// XEPNumber represents mam XEP number.
	XEPNumber = "0313"

	mamNamespace         = "urn:xmpp:mam:2"
	extendedMamNamespace = "urn:xmpp:mam:2#extended"

	archiveRequestedCtxKey = "mam:requested"
)

type archiveIDCtxKey int

const (
	sentArchiveIDKey archiveIDCtxKey = iota
	receivedArchiveIDKey
)

// Config contains mam module configuration options.
type Config struct {
	// QueueSize defines maximum number of archive messages stanzas.
	// When the limit is reached, the oldest message will be purged to make room for the new one.
	QueueSize int `fig:"queue_size" default:"1000"`
}

// Mam represents a mam (XEP-0313) module type.
type Mam struct {
	svc    *Service
	hk     *hook.Hooks
	router router.Router
	hosts  hosts
	logger kitlog.Logger
}

// New returns a new initialized mam instance.
func New(
	cfg Config,
	router router.Router,
	hosts *host.Hosts,
	rep repository.Repository,
	hk *hook.Hooks,
	logger kitlog.Logger,
) *Mam {
	logger = kitlog.With(logger, "module", ModuleName, "xep", XEPNumber)
	return &Mam{
		svc:    NewService(router, hk, rep, cfg.QueueSize, logger),
		router: router,
		hosts:  hosts,
		hk:     hk,
		logger: logger,
	}
}

// Name returns mam module name.
func (m *Mam) Name() string { return ModuleName }

// StreamFeature returns mam module stream feature.
func (m *Mam) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns mam server disco features.
func (m *Mam) ServerFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// AccountFeatures returns mam account disco features.
func (m *Mam) AccountFeatures(_ context.Context) ([]string, error) {
	return []string{mamNamespace, extendedMamNamespace}, nil
}

// Start starts mam module.
func (m *Mam) Start(_ context.Context) error {
	m.hk.AddHook(hook.C2SStreamMessageReceived, m.onMessageReceived, hook.HighestPriority)
	m.hk.AddHook(hook.S2SInStreamMessageReceived, m.onMessageReceived, hook.HighestPriority)

	m.hk.AddHook(hook.C2SStreamMessageRouted, m.onMessageRouted, hook.LowestPriority+2)
	m.hk.AddHook(hook.S2SInStreamMessageRouted, m.onMessageRouted, hook.LowestPriority+2)
	m.hk.AddHook(hook.UserDeleted, m.onUserDeleted, hook.DefaultPriority)

	level.Info(m.logger).Log("msg", "started mam module")
	return nil
}

// Stop stops mam module.
func (m *Mam) Stop(_ context.Context) error {
	m.hk.RemoveHook(hook.C2SStreamMessageReceived, m.onMessageReceived)
	m.hk.RemoveHook(hook.S2SInStreamMessageReceived, m.onMessageReceived)
	m.hk.RemoveHook(hook.C2SStreamMessageRouted, m.onMessageRouted)
	m.hk.RemoveHook(hook.S2SInStreamMessageRouted, m.onMessageRouted)
	m.hk.RemoveHook(hook.UserDeleted, m.onUserDeleted)

	level.Info(m.logger).Log("msg", "stopped mam module")
	return nil
}

// MatchesNamespace tells whether namespace matches mam module.
func (m *Mam) MatchesNamespace(namespace string, serverTarget bool) bool {
	if serverTarget {
		return false
	}
	return namespace == mamNamespace
}

// ProcessIQ process a mam iq.
func (m *Mam) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()

	if !fromJID.MatchesWithOptions(toJID, jid.MatchesBare) {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
		return nil
	}
	return m.svc.ProcessIQ(ctx, iq, func(_ string) error {
		fromJID := iq.FromJID()

		stm, err := m.router.C2S().LocalStream(fromJID.Node(), fromJID.Resource())
		if err != nil {
			return err
		}
		return stm.SetInfoValue(ctx, archiveRequestedCtxKey, true)
	})
}

func (m *Mam) onMessageReceived(execCtx *hook.ExecutionContext) error {
	var msg *stravaganza.Message

	switch inf := execCtx.Info.(type) {
	case *hook.C2SStreamInfo:
		msg = inf.Element.(*stravaganza.Message)
		inf.Element = m.addRecipientStanzaID(msg)
		execCtx.Info = inf

	case *hook.S2SStreamInfo:
		msg = inf.Element.(*stravaganza.Message)
		inf.Element = m.addRecipientStanzaID(msg)
		execCtx.Info = inf
	}
	return nil
}

func (m *Mam) onMessageRouted(execCtx *hook.ExecutionContext) error {
	var elem stravaganza.Element

	switch inf := execCtx.Info.(type) {
	case *hook.C2SStreamInfo:
		elem = inf.Element
	case *hook.S2SStreamInfo:
		elem = inf.Element
	}
	return m.handleRoutedMessage(execCtx, elem)
}

func (m *Mam) onUserDeleted(execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.UserInfo)
	return m.svc.DeleteArchive(execCtx.Context, inf.Username)
}

func (m *Mam) handleRoutedMessage(execCtx *hook.ExecutionContext, elem stravaganza.Element) error {
	msg, ok := elem.(*stravaganza.Message)
	if !ok {
		return nil
	}
	if !IsMessageArchievable(msg) {
		return nil
	}

	fromJID := msg.FromJID()
	if m.hosts.IsLocalHost(fromJID.Domain()) {
		sentArchiveID := uuid.New().String()
		archiveMsg := xmpputil.MakeStanzaIDMessage(msg, sentArchiveID, fromJID.ToBareJID().String())
		if err := m.svc.ArchiveMessage(execCtx.Context, archiveMsg, fromJID.ToBareJID().String(), sentArchiveID); err != nil {
			return err
		}
		execCtx.Context = context.WithValue(execCtx.Context, sentArchiveIDKey, sentArchiveID)
	}
	toJID := msg.ToJID()
	if !m.hosts.IsLocalHost(toJID.Domain()) {
		return nil
	}
	recievedArchiveID := xmpputil.MessageStanzaID(msg)
	if err := m.svc.ArchiveMessage(execCtx.Context, msg, toJID.ToBareJID().String(), recievedArchiveID); err != nil {
		return err
	}
	execCtx.Context = context.WithValue(execCtx.Context, receivedArchiveIDKey, recievedArchiveID)
	return nil
}

func (m *Mam) addRecipientStanzaID(originalMsg *stravaganza.Message) *stravaganza.Message {
	toJID := originalMsg.ToJID()
	if !m.hosts.IsLocalHost(toJID.Domain()) {
		return originalMsg
	}
	archiveID := uuid.New().String()
	return xmpputil.MakeStanzaIDMessage(originalMsg, archiveID, toJID.ToBareJID().String())
}

// IsArchiveRequested determines whether archive has been requested over a C2S stream by inspecting inf parameter.
func IsArchiveRequested(inf c2smodel.Info) bool {
	return inf.Bool(archiveRequestedCtxKey)
}

// ExtractSentArchiveID returns message sent archive ID by inspecting the passed context.
func ExtractSentArchiveID(ctx context.Context) string {
	ret, ok := ctx.Value(sentArchiveIDKey).(string)
	if ok {
		return ret
	}
	return ""
}

// ExtractReceivedArchiveID returns message received archive ID by inspecting the passed context.
func ExtractReceivedArchiveID(ctx context.Context) string {
	ret, ok := ctx.Value(receivedArchiveIDKey).(string)
	if ok {
		return ret
	}
	return ""
}
