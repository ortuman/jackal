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

package xep0191

import (
	"context"

	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	xmpputil "github.com/ortuman/jackal/util/xmpp"

	"github.com/jackal-xmpp/sonar"
	"github.com/ortuman/jackal/repository"

	"github.com/ortuman/jackal/event"

	"github.com/jackal-xmpp/stravaganza"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
)

const blockListNamespace = "urn:xmpp:blocking"

const (
	// ModuleName represents blocklist module name.
	ModuleName = "blocklist"

	// XEPNumber represents blocklist XEP number.
	XEPNumber = "0191"
)

// BlockList represents blocklist (XEP-0191) module type.
type BlockList struct {
	rep    repository.BlockList
	router router.Router
	sn     *sonar.Sonar
	subs   []sonar.SubID
}

// New returns a new initialized BlockList instance.
func New(router router.Router, rep repository.BlockList, sn *sonar.Sonar) *BlockList {
	return &BlockList{
		rep:    rep,
		router: router,
		sn:     sn,
	}
}

// Name returns blocklist module name.
func (m *BlockList) Name() string { return ModuleName }

// StreamFeature returns blocklist module stream feature.
func (m *BlockList) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns blocklist server disco features.
func (m *BlockList) ServerFeatures(_ context.Context) ([]string, error) {
	return []string{blockListNamespace}, nil
}

// AccountFeatures returns blocklist account disco features.
func (m *BlockList) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// MatchesNamespace tells whether namespace matches blocklist module.
func (m *BlockList) MatchesNamespace(namespace string, serverTarget bool) bool {
	if serverTarget {
		return false
	}
	return namespace == blockListNamespace
}

// ProcessIQ process a blocklist iq.
func (m *BlockList) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case iq.IsGet():
		return m.getBlockList(ctx, iq)
	case iq.IsSet():
		break
	}
	return nil
}

// Start starts blocklist module.
func (m *BlockList) Start(_ context.Context) error {
	m.subs = append(m.subs, m.sn.Subscribe(event.UserDeleted, m.onUserDeleted))

	log.Infow("Started blocklist module", "xep", XEPNumber)
	return nil
}

// Stop stops blocklist module.
func (m *BlockList) Stop(_ context.Context) error {
	for _, sub := range m.subs {
		m.sn.Unsubscribe(sub)
	}
	log.Infow("Stopped blocklist module", "xep", XEPNumber)
	return nil
}

func (m *BlockList) onUserDeleted(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.UserEventInfo)
	return m.rep.DeleteBlockListItems(ctx, inf.Username)
}

func (m *BlockList) getBlockList(ctx context.Context, iq *stravaganza.IQ) error {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()
	if fromJID.Node() != toJID.Node() {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
		return nil
	}
	if iq.ChildNamespace("blocklist", blockListNamespace) == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	blockList, err := m.rep.FetchBlockListItems(ctx, fromJID.Node())
	if err != nil {
		log.Errorw(err.Error(), "xep", XEPNumber)
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return nil
	}
	// send reply
	sb := stravaganza.NewBuilder("blocklist").
		WithAttribute(stravaganza.Namespace, blockListNamespace)
	for _, itm := range blockList {
		sb.WithChild(
			stravaganza.NewBuilder("item").
				WithAttribute("jid", itm.JID).
				Build(),
		)
	}

	resIQ := xmpputil.MakeResultIQ(iq, sb.Build())
	_, _ = m.router.Route(ctx, resIQ)
	return nil
}
