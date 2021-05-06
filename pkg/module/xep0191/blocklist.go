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
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/c2s"
	"github.com/ortuman/jackal/pkg/event"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	blocklistmodel "github.com/ortuman/jackal/pkg/model/blocklist"
	coremodel "github.com/ortuman/jackal/pkg/model/core"
	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	"github.com/ortuman/jackal/pkg/module"
	"github.com/ortuman/jackal/pkg/repository"
	"github.com/ortuman/jackal/pkg/router"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

const (
	requestedInfoKey = "xep0191:requested"

	blockListNamespace       = "urn:xmpp:blocking"
	blockListErrorsNamespace = "urn:xmpp:blocking:errors"

	blockedTargetErrorText = "Your active block list has denied the routing of this stanza."
)

const (
	// ModuleName represents blocklist module name.
	ModuleName = "blocklist"

	// XEPNumber represents blocklist XEP number.
	XEPNumber = "0191"
)

const (
	incomingIID int = iota
	outgoingIID
)

// BlockList represents blocklist (XEP-0191) module type.
type BlockList struct {
	rep    repository.Repository
	hosts  hosts
	router router.Router
	resMng resourceManager
	sn     *sonar.Sonar
	subs   []sonar.SubID
}

// New returns a new initialized BlockList instance.
func New(
	router router.Router,
	hosts *host.Hosts,
	resMng *c2s.ResourceManager,
	rep repository.Repository,
	sn *sonar.Sonar,
) *BlockList {
	return &BlockList{
		rep:    rep,
		router: router,
		hosts:  hosts,
		resMng: resMng,
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
	fromJID := iq.FromJID()
	toJID := iq.ToJID()
	if fromJID.Node() != toJID.Node() {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.Forbidden))
		return nil
	}
	switch {
	case iq.IsGet():
		return m.getBlockList(ctx, iq)
	case iq.IsSet():
		return m.alterBlockList(ctx, iq)
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

// Interceptors returns blocklist stanza interceptors.
func (m *BlockList) Interceptors() []module.StanzaInterceptor {
	return []module.StanzaInterceptor{
		{ID: incomingIID, Priority: math.MaxInt32, Incoming: true},
		{ID: outgoingIID, Priority: math.MaxInt32, Incoming: false},
	}
}

// InterceptStanza will be used by blocklist module to determine whether a stanza should be blocked.
func (m *BlockList) InterceptStanza(ctx context.Context, stanza stravaganza.Stanza, id int) (stravaganza.Stanza, error) {
	switch id {
	case incomingIID:
		return m.interceptIncomingStanza(ctx, stanza)
	case outgoingIID:
		return m.interceptOutgoingStanza(ctx, stanza)
	}
	return stanza, nil
}

func (m *BlockList) onUserDeleted(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.UserEventInfo)
	return m.rep.DeleteBlockListItems(ctx, inf.Username)
}

func (m *BlockList) interceptIncomingStanza(ctx context.Context, stanza stravaganza.Stanza) (stravaganza.Stanza, error) {
	fromJID := stanza.FromJID()
	toJID := stanza.ToJID()

	isLocalTo := m.hosts.IsLocalHost(toJID.Domain())
	if !isLocalTo || (isLocalTo && toJID.MatchesWithOptions(fromJID, jid.MatchesBare)) {
		return stanza, nil
	}
	bli, err := m.rep.FetchBlockListItems(ctx, toJID.Node())
	if err != nil {
		return nil, err
	}
	for _, itm := range bli {
		jd, _ := jid.NewWithString(itm.JID, true)
		if !jd.Matches(fromJID) {
			continue
		}
		// block stanza routing
		switch st := stanza.(type) {
		case *stravaganza.IQ:
			if st.IsGet() || st.IsSet() {
				_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(stanza, stanzaerror.ServiceUnavailable))
			}
		case *stravaganza.Message:
			_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(stanza, stanzaerror.ServiceUnavailable))
		}
		return nil, module.ErrInterceptStanzaInterrupted
	}
	return stanza, nil
}

func (m *BlockList) interceptOutgoingStanza(ctx context.Context, stanza stravaganza.Stanza) (stravaganza.Stanza, error) {
	fromJID := stanza.FromJID()
	toJID := stanza.ToJID()

	isLocalFrom := m.hosts.IsLocalHost(fromJID.Domain())
	if !isLocalFrom || (isLocalFrom && fromJID.MatchesWithOptions(toJID, jid.MatchesBare)) {
		return stanza, nil
	}
	bli, err := m.rep.FetchBlockListItems(ctx, fromJID.Node())
	if err != nil {
		return nil, err
	}
	for _, itm := range bli {
		jd, _ := jid.NewWithString(itm.JID, true)
		if !jd.Matches(toJID) {
			continue
		}
		// return <not-acceptable> stanza error
		se := stanzaerror.E(stanzaerror.NotAcceptable, stanza)
		se.Text = blockedTargetErrorText
		se.ApplicationElement = stravaganza.NewBuilder("blocked").
			WithAttribute(stravaganza.Namespace, blockListErrorsNamespace).
			Build()
		errStanza, _ := se.Stanza(false)

		_, _ = m.router.Route(ctx, errStanza)
		return nil, module.ErrInterceptStanzaInterrupted
	}
	return stanza, nil
}

func (m *BlockList) getBlockList(ctx context.Context, iq *stravaganza.IQ) error {
	if iq.ChildNamespace("blocklist", blockListNamespace) == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	fromJID := iq.FromJID()

	bli, err := m.rep.FetchBlockListItems(ctx, fromJID.Node())
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// send reply
	sb := stravaganza.NewBuilder("blocklist").
		WithAttribute(stravaganza.Namespace, blockListNamespace)
	for _, itm := range bli {
		sb.WithChild(
			stravaganza.NewBuilder("item").
				WithAttribute("jid", itm.JID).
				Build(),
		)
	}
	resIQ := xmpputil.MakeResultIQ(iq, sb.Build())
	_, _ = m.router.Route(ctx, resIQ)

	// mark as requested
	username := fromJID.Node()
	res := fromJID.Resource()

	stm := m.router.C2S().LocalStream(username, res)
	if stm == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return fmt.Errorf("xep0191: local stream not found: %s/%s", username, res)
	}
	if err := stm.SetInfoValue(ctx, requestedInfoKey, true); err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	log.Infow("Fetched blocklist", "username", username, "xep", XEPNumber)

	// post event
	var allJIDs []jid.JID
	for _, itm := range bli {
		j, _ := jid.NewWithString(itm.JID, false)
		allJIDs = append(allJIDs, *j)
	}
	return m.sn.Post(ctx, sonar.NewEventBuilder(event.BlockListFetched).
		WithInfo(&event.BlockListEventInfo{
			Username: username,
			JIDs:     allJIDs,
		}).
		WithSender(m).
		Build(),
	)
}

func (m *BlockList) alterBlockList(ctx context.Context, iq *stravaganza.IQ) error {
	// fetch current list
	bli, err := m.rep.FetchBlockListItems(ctx, iq.FromJID().Node())
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	if block := iq.ChildNamespace("block", blockListNamespace); block != nil {
		return m.blockJIDs(ctx, iq, block, bli)
	} else if unblock := iq.ChildNamespace("unblock", blockListNamespace); unblock != nil {
		return m.unblockJIDs(ctx, iq, unblock, bli)
	} else {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
}

func (m *BlockList) blockJIDs(ctx context.Context, iq *stravaganza.IQ, block stravaganza.Element, blockList []blocklistmodel.Item) error {
	// get JIDs
	js, err := getItemJIDs(block)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return err
	}
	// filter JIDs
	var blockJIDs []jid.JID

	for _, jd := range js {
		var found bool
		for _, bli := range blockList {
			if jd.String() == bli.JID {
				found = true
				break
			}
		}
		if !found {
			blockJIDs = append(blockJIDs, jd)
		}
	}
	username := iq.FromJID().Node()
	err = m.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		for _, bj := range blockJIDs {
			if err := tx.UpsertBlockListItem(ctx, &blocklistmodel.Item{
				Username: username,
				JID:      bj.String(),
			}); err != nil {
				return err
			}
			log.Infow("Blocked JID", "username", username, "jid", bj.String(), "xep", XEPNumber)
		}
		return nil
	})
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// send unavailable presences to blocked JIDs
	rss, err := m.resMng.GetResources(ctx, username)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	if err := m.sendUnavailablePresences(ctx, blockJIDs, rss, username); err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// send result IQ
	_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))

	// send block push
	m.sendPush(ctx, block, rss)

	// post event
	return m.sn.Post(ctx, sonar.NewEventBuilder(event.BlockListItemsBlocked).
		WithInfo(&event.BlockListEventInfo{
			Username: username,
			JIDs:     blockJIDs,
		}).
		WithSender(m).
		Build(),
	)
}

func (m *BlockList) unblockJIDs(ctx context.Context, iq *stravaganza.IQ, unblock stravaganza.Element, blockList []blocklistmodel.Item) error {
	// get JIDs
	js, err := getItemJIDs(unblock)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return err
	}
	var unblockJIDs []jid.JID
	if len(js) > 0 {
		// filter JIDs
		for _, jd := range js {
			var found bool
			for _, blItm := range blockList {
				if jd.String() == blItm.JID {
					found = true
					break
				}
			}
			if found {
				unblockJIDs = append(unblockJIDs, jd)
			}
		}
	} else {
		for _, bli := range blockList {
			jd, _ := jid.NewWithString(bli.JID, true)
			unblockJIDs = append(unblockJIDs, *jd)
		}
	}
	username := iq.FromJID().Node()
	err = m.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		for _, uj := range unblockJIDs {
			if err := tx.DeleteBlockListItem(ctx, &blocklistmodel.Item{
				Username: username,
				JID:      uj.String(),
			}); err != nil {
				return err
			}
			log.Infow("Unblocked JID", "username", username, "jid", uj.String(), "xep", XEPNumber)
		}
		return nil
	})
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// send available presences to unblocked JIDs
	rss, err := m.resMng.GetResources(ctx, username)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	if err := m.sendAvailablePresences(ctx, unblockJIDs, rss, username); err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// send result IQ
	_, _ = m.router.Route(ctx, xmpputil.MakeResultIQ(iq, nil))

	// send unblock push
	m.sendPush(ctx, unblock, rss)

	// post event
	return m.sn.Post(ctx, sonar.NewEventBuilder(event.BlockListItemsUnblocked).
		WithInfo(&event.BlockListEventInfo{
			Username: username,
			JIDs:     unblockJIDs,
		}).
		WithSender(m).
		Build(),
	)
}

func (m *BlockList) sendPush(ctx context.Context, pushed stravaganza.Element, resources []coremodel.Resource) {
	for _, res := range resources {
		if !res.Info.Bool(requestedInfoKey) { // block list requested?
			continue
		}
		pushIQ, _ := stravaganza.NewIQBuilder().
			WithAttribute(stravaganza.From, res.JID.ToBareJID().String()).
			WithAttribute(stravaganza.To, res.JID.String()).
			WithAttribute(stravaganza.Type, stravaganza.SetType).
			WithAttribute(stravaganza.ID, uuid.New().String()).
			WithChild(pushed).
			BuildIQ()

		_, _ = m.router.Route(ctx, pushIQ)
	}
}

func (m *BlockList) sendUnavailablePresences(ctx context.Context, blockJIDs []jid.JID, resources []coremodel.Resource, username string) error {
	targets, err := m.getPresenceTargets(ctx, blockJIDs, username)
	if err != nil {
		return err
	}
	for _, res := range resources {
		for _, target := range targets {
			pr := xmpputil.MakePresence(res.JID, &target, stravaganza.UnavailableType, nil)
			_, _ = m.router.Route(ctx, pr)
		}
	}
	return nil
}

func (m *BlockList) sendAvailablePresences(ctx context.Context, unblockJIDs []jid.JID, resources []coremodel.Resource, username string) error {
	targets, err := m.getPresenceTargets(ctx, unblockJIDs, username)
	if err != nil {
		return err
	}
	for _, res := range resources {
		for _, target := range targets {
			pr := xmpputil.MakePresence(res.JID, &target, stravaganza.AvailableType, res.Presence.AllChildren())
			_, _ = m.router.Route(ctx, pr)
		}
	}
	return nil
}

func (m *BlockList) getPresenceTargets(ctx context.Context, blockListJIDs []jid.JID, username string) ([]jid.JID, error) {
	ris, err := m.rep.FetchRosterItems(ctx, username)
	if err != nil {
		return nil, err
	}
	var targets []jid.JID

	for _, bj := range blockListJIDs {
		for _, ri := range ris {
			if ri.Subscription != rostermodel.From && ri.Subscription != rostermodel.Both {
				continue
			}
			rj, _ := jid.NewWithString(ri.JID, true)
			switch {
			case bj.IsFullWithUser() && bj.MatchesWithOptions(rj, jid.MatchesBare):
				targets = append(targets, bj)

			case bj.IsFullWithServer() && bj.MatchesWithOptions(rj, jid.MatchesDomain):
				t, _ := jid.New(rj.Node(), rj.Domain(), bj.Resource(), true)
				targets = append(targets, *t)

			case bj.IsBare() && bj.MatchesWithOptions(rj, jid.MatchesBare):
				fallthrough
			case bj.IsServer() && bj.MatchesWithOptions(rj, jid.MatchesDomain):
				targets = append(targets, *rj)
			}
		}
	}
	return targets, nil
}

func getItemJIDs(el stravaganza.Element) ([]jid.JID, error) {
	var retVal []jid.JID
	for _, itm := range el.Children("item") {
		j, err := jid.NewWithString(itm.Attribute("jid"), false)
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, *j)
	}
	return retVal, nil
}
