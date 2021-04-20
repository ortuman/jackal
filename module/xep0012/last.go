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
	"math"
	"strconv"
	"time"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/c2s"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	lastmodel "github.com/ortuman/jackal/model/last"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/repository"
	"github.com/ortuman/jackal/router"
	xmpputil "github.com/ortuman/jackal/util/xmpp"
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
	sn        *sonar.Sonar
	subs      []sonar.SubID
	startedAt int64
}

// New returns a new initialized Last instance.
func New(
	router router.Router,
	hosts *host.Hosts,
	resMng *c2s.ResourceManager,
	rep repository.Repository,
	sn *sonar.Sonar,
) *Last {
	return &Last{
		router: router,
		hosts:  hosts,
		resMng: resMng,
		rep:    rep,
		sn:     sn,
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

// Interceptors returns last activity stanza interceptor.
func (m *Last) Interceptors() []module.StanzaInterceptor {
	return []module.StanzaInterceptor{
		{Priority: math.MaxInt64, Incoming: true},
	}
}

// InterceptStanza will be used by last activity module to determine whether requesting entity is authorized.
func (m *Last) InterceptStanza(ctx context.Context, stanza stravaganza.Stanza, id int) (stravaganza.Stanza, error) {
	iq, ok := stanza.(*stravaganza.IQ)
	if !ok {
		return stanza, nil
	}
	toJID := iq.ToJID()
	if !m.hosts.IsLocalHost(toJID.Domain()) || !toJID.IsFullWithUser() || iq.ChildNamespace("query", lastActivityNamespace) == nil {
		return stanza, nil
	}
	ok, err := m.isSubscribedTo(ctx, toJID, iq.FromJID())
	if err != nil {
		return nil, err
	}
	if !ok {
		// reply on behalf
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
		return nil, module.ErrInterceptStanzaInterrupted
	}
	return stanza, nil
}

// Start starts last activity module.
func (m *Last) Start(_ context.Context) error {
	m.subs = append(m.subs, m.sn.Subscribe(event.UserDeleted, m.onUserDeleted))
	m.subs = append(m.subs, m.sn.Subscribe(event.C2SStreamPresenceReceived, m.onPresenceRecv))

	m.startedAt = time.Now().Unix()

	log.Infow("Started last module", "xep", XEPNumber)
	return nil
}

// Stop stops last activity module.
func (m *Last) Stop(_ context.Context) error {
	for _, sub := range m.subs {
		m.sn.Unsubscribe(sub)
	}
	log.Infow("Stopped last module", "xep", XEPNumber)
	return nil
}

func (m *Last) onUserDeleted(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.UserEventInfo)
	return m.rep.DeleteLast(ctx, inf.Username)
}

func (m *Last) onPresenceRecv(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.C2SStreamEventInfo)
	pr := inf.Stanza.(*stravaganza.Presence)
	return m.processPresence(ctx, pr)
}

func (m *Last) processPresence(ctx context.Context, pr *stravaganza.Presence) error {
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
	return nil
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
	return nil
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
