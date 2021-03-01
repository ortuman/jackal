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

package xep0115

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/repository"
	"github.com/ortuman/jackal/router"
)

const (
	capabilitiesFeature = "http://jabber.org/protocol/caps"

	discoInfoNamespace = "http://jabber.org/protocol/disco#info"
)

type nodeVer struct {
	node string
	ver  string
}

const (
	// ModuleName represents entity capabilities module name.
	ModuleName = "caps"

	// XEPNumber represents entity capabilities XEP number.
	XEPNumber = "0115"
)

// Capabilities represents entity capabilities module type.
type Capabilities struct {
	mu     sync.RWMutex
	reqs   map[string]nodeVer
	router router.Router
	rep    repository.Capabilities
	sn     *sonar.Sonar
	subs   []sonar.SubID
}

// New creates and initializes a new Capabilities instance.
func New(
	router router.Router,
	rep repository.Capabilities,
	sn *sonar.Sonar,
) *Capabilities {
	return &Capabilities{
		reqs:   make(map[string]nodeVer),
		router: router,
		rep:    rep,
		sn:     sn,
	}
}

// Name returns entity capabilities module name.
func (m *Capabilities) Name() string { return ModuleName }

// StreamFeature returns entity capabilities module stream feature.
func (m *Capabilities) StreamFeature() stravaganza.Element { return nil }

// ServerFeatures returns entity capabilities module server disco features.
func (m *Capabilities) ServerFeatures() []string { return []string{capabilitiesFeature} }

// AccountFeatures returns entity capabilities module account disco features.
func (m *Capabilities) AccountFeatures() []string { return []string{capabilitiesFeature} }

// Start starts entity capabilities module.
func (m *Capabilities) Start(_ context.Context) error {
	m.subs = append(m.subs, m.sn.Subscribe(event.C2SStreamPresenceReceived, m.onC2SPresenceRecv))
	m.subs = append(m.subs, m.sn.Subscribe(event.S2SInStreamPresenceReceived, m.onS2SPresenceRecv))

	log.Infow("Started capabilities module", "xep", XEPNumber)
	return nil
}

// Stop stops entity capabilities module.
func (m *Capabilities) Stop(_ context.Context) error {
	for _, sub := range m.subs {
		m.sn.Unsubscribe(sub)
	}
	log.Infow("Stopped capabilities module", "xep", XEPNumber)
	return nil
}

func (m *Capabilities) onC2SPresenceRecv(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.C2SStreamEventInfo)
	pr := inf.Stanza.(*stravaganza.Presence)
	return m.processPresence(ctx, pr)
}

func (m *Capabilities) onS2SPresenceRecv(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.S2SStreamEventInfo)
	pr := inf.Stanza.(*stravaganza.Presence)
	return m.processPresence(ctx, pr)
}

func (m *Capabilities) processPresence(ctx context.Context, pr *stravaganza.Presence) error {
	if pr.ToJID().IsFull() {
		return nil
	}
	caps := pr.ChildNamespace("c", capabilitiesFeature)
	if caps == nil {
		return nil
	}
	node := caps.Attribute("node")
	ver := caps.Attribute("ver")

	// fetch registered capabilities
	exist, err := m.rep.CapabilitiesExist(ctx, node, ver)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}
	return m.requestDiscoInfo(ctx, pr.FromJID(), pr.ToJID(), node, ver)
}

func (m *Capabilities) requestDiscoInfo(ctx context.Context, fromJID, toJID *jid.JID, node, ver string) error {
	reqID := uuid.New().String()

	m.mu.Lock()
	m.reqs[reqID] = nodeVer{node: node, ver: ver}
	m.mu.Unlock()

	discoIQ, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, reqID).
		WithAttribute(stravaganza.From, toJID.String()).
		WithAttribute(stravaganza.To, fromJID.String()).
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, discoInfoNamespace).
				WithAttribute("node", fmt.Sprintf("%s#%s", node, ver)).
				Build(),
		).
		BuildIQ(false)
	return m.router.Route(ctx, discoIQ)
}
