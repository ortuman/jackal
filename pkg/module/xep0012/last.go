// Copyright 2021 The jackal Authors
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

package xep0012

import (
	"context"
	"strconv"
	"time"

	hook2 "github.com/ortuman/jackal/pkg/hook"

	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/c2s"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	lastmodel "github.com/ortuman/jackal/pkg/model/last"
	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	"github.com/ortuman/jackal/pkg/repository"
	"github.com/ortuman/jackal/pkg/router"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

const lastActivityNamespace = "jabber:iq:last"

const (
	// ModuleName represents last activity module name.
	ModuleName = "last"

	// XEPNumber represents last activity XEP number.
	XEPNumber = "0012"
)

// Last represents a last activity (XEP-0012) module type.
type Last struct {
	router    router.Router
	hosts     hosts
	resMng    resourceManager
	rep       repository.Repository
	hk        *hook2.Hooks
	startedAt int64
}

// New returns a new initialized Last instance.
func New(
	router router.Router,
	hosts *host.Hosts,
	resMng *c2s.ResourceManager,
	rep repository.Repository,
	hk *hook2.Hooks,
) *Last {
	return &Last{
		router: router,
		hosts:  hosts,
		resMng: resMng,
		rep:    rep,
		hk:     hk,
	}
}

// Name returns last activity module name.
func (m *Last) Name() string { return ModuleName }

// StreamFeature returns last activity stream feature.
func (m *Last) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns server last activity features.
func (m *Last) ServerFeatures(_ context.Context) ([]string, error) {
	return []string{lastActivityNamespace}, nil
}

// AccountFeatures returns account last activity features.
func (m *Last) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// MatchesNamespace tells whether namespace matches last activity module.
func (m *Last) MatchesNamespace(namespace string, _ bool) bool {
	return namespace == lastActivityNamespace
}

// ProcessIQ process a last activity info iq.
func (m *Last) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case iq.IsGet() && iq.ChildNamespace("query", lastActivityNamespace) != nil:
		return m.getLastActivity(ctx, iq)
	default:
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
}

// Start starts last activity module.
func (m *Last) Start(_ context.Context) error {
	m.hk.AddHook(hook2.C2SStreamElementReceived, m.onElementRecv, hook2.DefaultPriority)
	m.hk.AddHook(hook2.S2SInStreamElementReceived, m.onElementRecv, hook2.DefaultPriority)
	m.hk.AddHook(hook2.C2SStreamPresenceReceived, m.onC2SPresenceRecv, hook2.DefaultPriority)
	m.hk.AddHook(hook2.UserDeleted, m.onUserDeleted, hook2.DefaultPriority)

	m.startedAt = time.Now().Unix()

	log.Infow("Started last module", "xep", XEPNumber)
	return nil
}

// Stop stops last activity module.
func (m *Last) Stop(_ context.Context) error {
	m.hk.RemoveHook(hook2.C2SStreamElementReceived, m.onElementRecv)
	m.hk.RemoveHook(hook2.S2SInStreamElementReceived, m.onElementRecv)
	m.hk.RemoveHook(hook2.C2SStreamPresenceReceived, m.onC2SPresenceRecv)
	m.hk.RemoveHook(hook2.UserDeleted, m.onUserDeleted)

	log.Infow("Stopped last module", "xep", XEPNumber)
	return nil
}

func (m *Last) onElementRecv(ctx context.Context, execCtx *hook2.ExecutionContext) (halt bool, err error) {
	var iq *stravaganza.IQ
	var ok bool

	switch inf := execCtx.Info.(type) {
	case *hook2.C2SStreamHookInfo:
		iq, ok = inf.Element.(*stravaganza.IQ)
	case *hook2.S2SStreamHookInfo:
		iq, ok = inf.Element.(*stravaganza.IQ)
	default:
		return false, nil
	}
	if !ok {
		return false, nil
	}
	return m.processIncomingIQ(ctx, iq)
}

func (m *Last) processIncomingIQ(ctx context.Context, iq *stravaganza.IQ) (halt bool, err error) {
	toJID := iq.ToJID()

	isLocalTo := m.hosts.IsLocalHost(toJID.Domain())
	if !isLocalTo || !toJID.IsFullWithUser() || iq.ChildNamespace("query", lastActivityNamespace) == nil {
		return false, nil
	}
	ok, err := m.isSubscribedTo(ctx, toJID, iq.FromJID())
	if err != nil {
		return false, err
	}
	if !ok {
		// reply on behalf
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
		return true, nil // already handled
	}
	return false, nil
}

func (m *Last) onUserDeleted(ctx context.Context, execCtx *hook2.ExecutionContext) (halt bool, err error) {
	inf := execCtx.Info.(*hook2.UserHookInfo)
	return false, m.rep.DeleteLast(ctx, inf.Username)
}

func (m *Last) onC2SPresenceRecv(ctx context.Context, execCtx *hook2.ExecutionContext) (halt bool, err error) {
	inf := execCtx.Info.(*hook2.C2SStreamHookInfo)
	pr := inf.Element.(*stravaganza.Presence)
	return false, m.processC2SPresence(ctx, pr)
}

func (m *Last) processC2SPresence(ctx context.Context, pr *stravaganza.Presence) error {
	fromJID := pr.FromJID()
	toJID := pr.ToJID()
	if !pr.IsUnavailable() || !toJID.IsBare() || fromJID.Node() != toJID.Node() {
		return nil
	}
	username := fromJID.Node()
	err := m.rep.UpsertLast(ctx, &lastmodel.Last{
		Username: username,
		Seconds:  time.Now().Unix(),
		Status:   pr.Status(),
	})
	if err != nil {
		return err
	}
	log.Infow("Last activity registered", "username", username, "xep", XEPNumber)
	return nil
}

func (m *Last) getLastActivity(ctx context.Context, iq *stravaganza.IQ) error {
	if iq.ToJID().IsServer() {
		return m.getServerLastActivity(ctx, iq)
	}
	return m.getAccountLastActivity(ctx, iq)
}

func (m *Last) getServerLastActivity(ctx context.Context, iq *stravaganza.IQ) error {
	// reply with server uptime
	m.sendReply(ctx, iq, time.Now().Unix()-m.startedAt, "")

	log.Infow("Sent server uptime", "username", iq.FromJID().Node(), "xep", XEPNumber)

	_, err := m.hk.Run(ctx, hook2.LastActivityFetched, &hook2.ExecutionContext{
		Info: &hook2.LastActivityHookInfo{
			Username: iq.FromJID().Node(),
			JID:      iq.ToJID(),
		},
		Sender: m,
	})
	return err
}

func (m *Last) getAccountLastActivity(ctx context.Context, iq *stravaganza.IQ) error {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()
	ok, err := m.isSubscribedTo(ctx, toJID, fromJID)
	if err != nil {
		return err
	}
	if !ok {
		// requesting entity is not authorized
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
		return nil
	}
	rss, err := m.resMng.GetResources(ctx, toJID.Node())
	if err != nil {
		return err
	}
	if len(rss) > 0 {
		// online user
		m.sendReply(ctx, iq, 0, "")
		return nil
	}
	lst, err := m.rep.FetchLast(ctx, toJID.Node())
	if err != nil {
		return err
	}
	if lst == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.ItemNotFound))
		return nil
	}
	m.sendReply(ctx, iq, time.Now().Unix()-lst.Seconds, lst.Status)

	log.Infow("Sent last activity", "username", fromJID.Node(), "target", toJID.Node(), "xep", XEPNumber)

	_, err = m.hk.Run(ctx, hook2.LastActivityFetched, &hook2.ExecutionContext{
		Info: &hook2.LastActivityHookInfo{
			Username: fromJID.Node(),
			JID:      toJID,
		},
		Sender: m,
	})
	return err
}

func (m *Last) sendReply(ctx context.Context, iq *stravaganza.IQ, seconds int64, status string) {
	resIQ := xmpputil.MakeResultIQ(iq, stravaganza.NewBuilder("query").
		WithAttribute(stravaganza.Namespace, lastActivityNamespace).
		WithAttribute("seconds", strconv.FormatInt(seconds, 10)).
		WithText(status).
		Build(),
	)
	_, _ = m.router.Route(ctx, resIQ)
}

func (m *Last) isSubscribedTo(ctx context.Context, contactJID *jid.JID, userJID *jid.JID) (bool, error) {
	if contactJID.MatchesWithOptions(userJID, jid.MatchesBare) {
		return true, nil
	}
	ri, err := m.rep.FetchRosterItem(ctx, contactJID.Node(), userJID.ToBareJID().String())
	if err != nil {
		return false, err
	}
	return ri != nil && (ri.Subscription == rostermodel.From || ri.Subscription == rostermodel.Both), nil
}
