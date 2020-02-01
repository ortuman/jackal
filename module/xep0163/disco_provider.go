/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"context"

	"github.com/ortuman/jackal/log"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

var pepFeatures = []string{
	"http://jabber.org/protocol/pubsub#access-presence",
	"http://jabber.org/protocol/pubsub#auto-create",
	"http://jabber.org/protocol/pubsub#auto-subscribe",
	"http://jabber.org/protocol/pubsub#config-node",
	"http://jabber.org/protocol/pubsub#create-and-configure",
	"http://jabber.org/protocol/pubsub#create-nodes",
	"http://jabber.org/protocol/pubsub#filtered-notifications",
	"http://jabber.org/protocol/pubsub#persistent-items",
	"http://jabber.org/protocol/pubsub#publish",
	"http://jabber.org/protocol/pubsub#retrieve-items",
	"http://jabber.org/protocol/pubsub#subscribe",
}

type discoInfoProvider struct {
	rosterRep repository.Roster
	pubSubRep repository.PubSub
}

func (p *discoInfoProvider) Identities(_ context.Context, _, _ *jid.JID, node string) []xep0030.Identity {
	var identities []xep0030.Identity
	if len(node) > 0 {
		identities = append(identities, xep0030.Identity{Type: "leaf", Category: "pubsub"})
	} else {
		identities = append(identities, xep0030.Identity{Type: "collection", Category: "pubsub"})
	}
	identities = append(identities, xep0030.Identity{Type: "pep", Category: "pubsub"})
	return identities
}

func (p *discoInfoProvider) Features(_ context.Context, _, _ *jid.JID, _ string) ([]xep0030.Feature, *xmpp.StanzaError) {
	return pepFeatures, nil
}

func (p *discoInfoProvider) Form(_ context.Context, _, _ *jid.JID, _ string) (*xep0004.DataForm, *xmpp.StanzaError) {
	return nil, nil
}

func (p *discoInfoProvider) Items(ctx context.Context, toJID, fromJID *jid.JID, node string) ([]xep0030.Item, *xmpp.StanzaError) {
	if !p.isSubscribedTo(ctx, toJID, fromJID) {
		return nil, xmpp.ErrSubscriptionRequired
	}
	host := toJID.ToBareJID().String()

	if len(node) > 0 {
		// return node items
		return p.nodeItems(ctx, host, node)
	}
	// return host nodes
	return p.hostNodes(ctx, host)
}

func (p *discoInfoProvider) hostNodes(ctx context.Context, host string) ([]xep0030.Item, *xmpp.StanzaError) {
	var items []xep0030.Item

	nodes, err := p.pubSubRep.FetchNodes(ctx, host)
	if err != nil {
		log.Error(err)
		return nil, xmpp.ErrInternalServerError
	}
	for _, node := range nodes {
		items = append(items, xep0030.Item{
			Jid:  host,
			Node: node.Name,
			Name: node.Options.Title,
		})
	}
	return items, nil
}

func (p *discoInfoProvider) nodeItems(ctx context.Context, host, node string) ([]xep0030.Item, *xmpp.StanzaError) {
	var items []xep0030.Item

	n, err := p.pubSubRep.FetchNode(ctx, host, node)
	if err != nil {
		log.Error(err)
		return nil, xmpp.ErrInternalServerError
	}
	if n == nil {
		// does not exist
		return nil, xmpp.ErrItemNotFound
	}
	nodeItems, err := p.pubSubRep.FetchNodeItems(ctx, host, node)
	if err != nil {
		log.Error(err)
		return nil, xmpp.ErrInternalServerError
	}
	for _, nodeItem := range nodeItems {
		items = append(items, xep0030.Item{
			Jid:  nodeItem.Publisher,
			Name: nodeItem.ID,
		})
	}
	return items, nil
}

func (p *discoInfoProvider) isSubscribedTo(ctx context.Context, contact *jid.JID, userJID *jid.JID) bool {
	if contact.MatchesWithOptions(userJID, jid.MatchesBare) {
		return true
	}
	ri, err := p.rosterRep.FetchRosterItem(ctx, userJID.Node(), contact.ToBareJID().String())
	if err != nil {
		log.Error(err)
		return false
	}
	if ri == nil {
		return false
	}
	return ri.Subscription == rostermodel.SubscriptionTo || ri.Subscription == rostermodel.SubscriptionBoth
}
