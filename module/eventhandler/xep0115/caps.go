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
	"time"

	discomodel "github.com/ortuman/jackal/model/disco"

	"github.com/ortuman/jackal/module/xep0004"

	capsmodel "github.com/ortuman/jackal/model/caps"

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
	formsNamespace     = "jabber:x:data"
)

type capsInfo struct {
	hash string
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
	reqs   map[string]capsInfo
	clrTms map[string]*time.Timer
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
		reqs:   make(map[string]capsInfo),
		clrTms: make(map[string]*time.Timer),
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
	m.subs = append(m.subs, m.sn.Subscribe(event.C2SStreamIQReceived, m.onC2SIQRecv))
	m.subs = append(m.subs, m.sn.Subscribe(event.S2SInStreamIQReceived, m.onS2SIQRecv))

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

func (m *Capabilities) onC2SIQRecv(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.C2SStreamEventInfo)
	iq := inf.Stanza.(*stravaganza.IQ)
	return m.processIQ(ctx, iq)
}

func (m *Capabilities) onS2SIQRecv(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.S2SStreamEventInfo)
	iq := inf.Stanza.(*stravaganza.IQ)
	return m.processIQ(ctx, iq)
}

func (m *Capabilities) processPresence(ctx context.Context, pr *stravaganza.Presence) error {
	if pr.ToJID().IsFull() {
		return nil
	}
	caps := pr.ChildNamespace("c", capabilitiesFeature)
	if caps == nil {
		return nil
	}
	ci := capsInfo{
		hash: caps.Attribute("hash"),
		node: caps.Attribute("node"),
		ver:  caps.Attribute("ver"),
	}
	// fetch registered capabilities
	exist, err := m.rep.CapabilitiesExist(ctx, ci.node, ci.ver)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}
	m.requestDiscoInfo(ctx, pr.FromJID(), pr.ToJID(), ci)
	return nil
}

func (m *Capabilities) processIQ(ctx context.Context, iq *stravaganza.IQ) error {
	reqID := iq.Attribute(stravaganza.ID)

	m.mu.Lock()
	if tm := m.clrTms[reqID]; tm != nil {
		tm.Stop()
	}
	nv, ok := m.reqs[reqID]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()
	if err := m.processDiscoInfo(ctx, iq, nv); err != nil {
		log.Warnw(fmt.Sprintf("Failed to verify disco info: %v", err), "xep", XEPNumber)
	}
	return nil
}

func (m *Capabilities) requestDiscoInfo(ctx context.Context, fromJID, toJID *jid.JID, ci capsInfo) {
	reqID := uuid.New().String()

	m.mu.Lock()
	m.reqs[reqID] = ci
	m.clrTms[reqID] = time.AfterFunc(time.Minute, func() {
		m.clearPendingReq(reqID) // discard pending request
	})
	m.mu.Unlock()

	discoIQ, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, reqID).
		WithAttribute(stravaganza.From, toJID.String()).
		WithAttribute(stravaganza.To, fromJID.String()).
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, discoInfoNamespace).
				WithAttribute("node", fmt.Sprintf("%s#%s", ci.node, ci.ver)).
				Build(),
		).
		BuildIQ(false)

	_ = m.router.Route(ctx, discoIQ)
}

func (m *Capabilities) processDiscoInfo(ctx context.Context, iq *stravaganza.IQ, ci capsInfo) error {
	dq := iq.ChildNamespace("query", discoInfoNamespace)
	if dq == nil {
		return nil
	}
	var err error

	var idns []discomodel.Identity
	var fs []discomodel.Feature
	var form *xep0004.DataForm

	// get identities
	for _, idnEl := range dq.Children("identity") {
		idns = append(idns, discomodel.Identity{
			Category: idnEl.Attribute("category"),
			Name:     idnEl.Attribute("name"),
			Type:     idnEl.Attribute("type"),
			Lang:     idnEl.Attribute(stravaganza.Language),
		})
	}
	// get features
	for _, featureEl := range dq.Children("feature") {
		fs = append(fs, featureEl.Attribute("var"))
	}
	// get form
	if formEl := dq.ChildNamespace("x", formsNamespace); formEl != nil {
		form, err = xep0004.NewFormFromElement(formEl)
		if err != nil {
			return nil
		}
	}
	ver := computeVerification(idns, fs, form)
	if ver != ci.ver {
		return fmt.Errorf("xep0115: verification string mismatch: got %s, expected %s", ver, ci.ver)
	}
	return m.rep.UpsertCapabilities(ctx, &capsmodel.Capabilities{
		Node:     ci.node,
		Ver:      ci.ver,
		Features: fs,
	})
}

func (m *Capabilities) clearPendingReq(reqID string) {
	m.mu.Lock()
	delete(m.reqs, reqID)
	delete(m.clrTms, reqID)
	m.mu.Unlock()
}
