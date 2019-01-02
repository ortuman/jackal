/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type serverProvider struct {
	router          *router.Router
	mu              sync.RWMutex
	serverItems     []Item
	serverFeatures  []Feature
	accountFeatures []Feature
}

func (sp *serverProvider) Identities(toJID, fromJID *jid.JID, node string) []Identity {
	if node != "" {
		return nil
	}
	if toJID.IsServer() {
		return []Identity{{Type: "im", Category: "server", Name: "jackal"}}
	}
	return []Identity{{Type: "registered", Category: "account"}}
}

func (sp *serverProvider) Items(toJID, fromJID *jid.JID, node string) ([]Item, *xmpp.StanzaError) {
	if node != "" {
		return nil, nil
	}
	var itms []Item
	if toJID.IsServer() {
		itms = append(itms, Item{Jid: fromJID.ToBareJID().String()})
		itms = append(itms, sp.serverItems...)
	} else {
		// add account resources
		if sp.isSubscribedTo(toJID, fromJID) {
			stms := sp.router.UserStreams(toJID.Node())
			for _, stm := range stms {
				itms = append(itms, Item{Jid: stm.JID().String()})
			}
		} else {
			return nil, xmpp.ErrSubscriptionRequired
		}
	}
	return itms, nil
}

func (sp *serverProvider) Features(toJID, fromJID *jid.JID, node string) ([]Feature, *xmpp.StanzaError) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	if node != "" {
		return nil, nil
	}
	if toJID.IsServer() {
		return sp.serverFeatures, nil
	}
	if sp.isSubscribedTo(toJID, fromJID) {
		return sp.accountFeatures, nil
	}
	return nil, xmpp.ErrSubscriptionRequired
}

func (sp *serverProvider) Form(toJID, fromJID *jid.JID, node string) (*xep0004.DataForm, *xmpp.StanzaError) {
	return nil, nil
}

func (sp *serverProvider) registerServerItem(item Item) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	for _, itm := range sp.serverItems {
		if itm.Jid == item.Jid && itm.Name == item.Name && itm.Node == item.Node {
			return // already registered
		}
	}
	sp.serverItems = append(sp.serverItems, item)
}

func (sp *serverProvider) unregisterServerItem(item Item) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	for i, itm := range sp.serverItems {
		if itm.Jid == item.Jid && itm.Name == item.Name && itm.Node == item.Node {
			sp.serverItems = append(sp.serverItems[:i], sp.serverItems[i+1:]...)
			return
		}
	}
}

func (sp *serverProvider) registerServerFeature(feature Feature) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	for _, f := range sp.serverFeatures {
		if f == feature {
			return
		}
	}
	sp.serverFeatures = append(sp.serverFeatures, feature)
}

func (sp *serverProvider) unregisterServerFeature(feature Feature) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	for i, f := range sp.serverFeatures {
		if f == feature {
			sp.serverFeatures = append(sp.serverFeatures[:i], sp.serverFeatures[i+1:]...)
			return
		}
	}
}

func (sp *serverProvider) registerAccountFeature(feature Feature) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	for _, f := range sp.accountFeatures {
		if f == feature {
			return
		}
	}
	sp.accountFeatures = append(sp.accountFeatures, feature)
}

func (sp *serverProvider) unregisterAccountFeature(feature Feature) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	for i, f := range sp.accountFeatures {
		if f == feature {
			sp.accountFeatures = append(sp.accountFeatures[:i], sp.accountFeatures[i+1:]...)
			return
		}
	}
}

func (sp *serverProvider) isSubscribedTo(contact *jid.JID, userJID *jid.JID) bool {
	if contact.Matches(userJID, jid.MatchesBare) {
		return true
	}
	ri, err := storage.FetchRosterItem(userJID.Node(), contact.ToBareJID().String())
	if err != nil {
		log.Error(err)
		return false
	}
	if ri == nil {
		return false
	}
	return ri.Subscription == rostermodel.SubscriptionTo || ri.Subscription == rostermodel.SubscriptionBoth
}
