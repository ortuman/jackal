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
	"strconv"

	coremodel "github.com/ortuman/jackal/model/core"

	"github.com/google/uuid"

	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/c2s"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/log"
	blocklistmodel "github.com/ortuman/jackal/model/blocklist"
	"github.com/ortuman/jackal/repository"
	"github.com/ortuman/jackal/router"
	xmpputil "github.com/ortuman/jackal/util/xmpp"
)

const (
	blockListRequestedCtxKey = "blocklist:requested"

	blockListNamespace = "urn:xmpp:blocking"
)

const (
	// ModuleName represents blocklist module name.
	ModuleName = "blocklist"

	// XEPNumber represents blocklist XEP number.
	XEPNumber = "0191"
)

// BlockList represents blocklist (XEP-0191) module type.
type BlockList struct {
	rep    repository.Repository
	router router.Router
	resMng resourceManager
	sn     *sonar.Sonar
	subs   []sonar.SubID
}

// New returns a new initialized BlockList instance.
func New(router router.Router, resMng *c2s.ResourceManager, rep repository.Repository, sn *sonar.Sonar) *BlockList {
	return &BlockList{
		rep:    rep,
		router: router,
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

func (m *BlockList) onUserDeleted(ctx context.Context, ev sonar.Event) error {
	inf := ev.Info().(*event.UserEventInfo)
	return m.rep.DeleteBlockListItems(ctx, inf.Username)
}

func (m *BlockList) getBlockList(ctx context.Context, iq *stravaganza.IQ) error {
	if iq.ChildNamespace("blocklist", blockListNamespace) == nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
	fromJID := iq.FromJID()
	blockList, err := m.rep.FetchBlockListItems(ctx, fromJID.Node())
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
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

	// mark as requested
	return m.setStreamValue(ctx, fromJID.Node(), fromJID.Resource(), blockListRequestedCtxKey, strconv.FormatBool(true))
}

func (m *BlockList) alterBlockList(ctx context.Context, iq *stravaganza.IQ) error {
	// fetch current list
	blockList, err := m.rep.FetchBlockListItems(ctx, iq.FromJID().Node())
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	if block := iq.ChildNamespace("block", blockListNamespace); block != nil {
		return m.blockJIDs(ctx, iq, block, blockList)
	} else if unblock := iq.ChildNamespace("unblock", blockListNamespace); unblock != nil {
		return m.unblockJIDs(ctx, iq, unblock, blockList)
	} else {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
}

func (m *BlockList) blockJIDs(ctx context.Context, iq *stravaganza.IQ, block stravaganza.Element, blockList []blocklistmodel.Item) error {
	username := iq.FromJID().Node()

	// get blocklist items
	items, err := getItems(block, username)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return err
	}
	// filter items
	var blockItems []blocklistmodel.Item

	for _, itm := range items {
		var found bool
		for _, blItm := range blockList {
			if itm.JID == blItm.JID {
				found = true
				break
			}
		}
		if !found {
			blockItems = append(blockItems, itm)
		}
	}
	err = m.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		for _, itm := range blockItems {
			if err := tx.UpsertBlockListItem(ctx, &itm); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// send unavailable presence to blocked JIDs
	if err := m.sendUnavailablePresences(ctx, blockItems, username); err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// send block push
	rss, err := m.resMng.GetResources(ctx, username)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	m.sendPush(ctx, block, rss)
	return nil
}

func (m *BlockList) unblockJIDs(ctx context.Context, iq *stravaganza.IQ, unblock stravaganza.Element, blockList []blocklistmodel.Item) error {
	username := iq.FromJID().Node()

	// get blocklist items
	items, err := getItems(unblock, username)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return err
	}
	var unblockItems []blocklistmodel.Item
	if len(items) > 0 {
		// filter items
		for _, itm := range items {
			var found bool
			for _, blItm := range blockList {
				if itm.JID == blItm.JID {
					found = true
					break
				}
			}
			if found {
				unblockItems = append(unblockItems, itm)
			}
		}
	} else {
		unblockItems = blockList
	}
	err = m.rep.InTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		for _, itm := range unblockItems {
			if err := tx.DeleteBlockListItem(ctx, &itm); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// send available presence
	rss, err := m.resMng.GetResources(ctx, username)
	if err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	if err := m.sendAvailablePresences(ctx, unblockItems, rss, username); err != nil {
		_, _ = m.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.InternalServerError))
		return err
	}
	// send unblock push
	m.sendPush(ctx, unblock, rss)
	return nil
}

func (m *BlockList) sendPush(ctx context.Context, push stravaganza.Element, resources []coremodel.Resource) {
	for _, res := range resources {
		// blocklist requested?
		ok, _ := strconv.ParseBool(res.Value(blockListRequestedCtxKey))
		if !ok {
			continue
		}
		pushIQ, _ := stravaganza.NewIQBuilder().
			WithAttribute(stravaganza.From, res.JID.ToBareJID().String()).
			WithAttribute(stravaganza.To, res.JID.String()).
			WithAttribute(stravaganza.Type, stravaganza.SetType).
			WithAttribute(stravaganza.ID, uuid.New().String()).
			WithChild(push).
			BuildIQ(false)

		_, _ = m.router.Route(ctx, pushIQ)
	}
}

func (m *BlockList) sendUnavailablePresences(ctx context.Context, blockedItems []blocklistmodel.Item, username string) error {
	return nil
}

func (m *BlockList) sendAvailablePresences(ctx context.Context, unblockedItems []blocklistmodel.Item, resources []coremodel.Resource, username string) error {
	return nil
}

func (m *BlockList) setStreamValue(ctx context.Context, username, resource, key, val string) error {
	stm := m.router.C2S().LocalStream(username, resource)
	if stm == nil {
		return errStreamNotFound(username, resource)
	}
	return stm.SetValue(ctx, key, val)
}

func getItems(el stravaganza.Element, username string) ([]blocklistmodel.Item, error) {
	var retVal []blocklistmodel.Item
	for _, itm := range el.Children("item") {
		j, err := jid.NewWithString(itm.Attribute("jid"), false)
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, blocklistmodel.Item{
			Username: username,
			JID:      j.String(),
		})
	}
	return retVal, nil
}

func errStreamNotFound(username, resource string) error {
	return fmt.Errorf("roster: local stream not found: %s/%s", username, resource)
}
