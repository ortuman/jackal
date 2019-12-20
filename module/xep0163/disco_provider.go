package xep0163

import (
	"github.com/ortuman/jackal/log"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type discoInfoProvider struct {
	host string
}

func (p *discoInfoProvider) Items(toJID, fromJID *jid.JID, node string) ([]xep0030.Item, *xmpp.StanzaError) {
	if !p.isSubscribedTo(toJID, fromJID) {
		return nil, xmpp.ErrSubscriptionRequired
	}
	if len(node) > 0 {
		// return node items
		return p.nodeItems(node)
	}
	// return host nodes
	return p.hostNodes()
}

func (p *discoInfoProvider) hostNodes() ([]xep0030.Item, *xmpp.StanzaError) {
	var items []xep0030.Item

	nodes, err := storage.FetchNodes(p.host)
	if err != nil {
		log.Error(err)
		return nil, xmpp.ErrInternalServerError
	}
	for _, node := range nodes {
		items = append(items, xep0030.Item{
			Jid:  p.host,
			Node: node.Name,
			Name: node.Options.Title,
		})
	}
	return items, nil
}

func (p *discoInfoProvider) nodeItems(node string) ([]xep0030.Item, *xmpp.StanzaError) {
	var items []xep0030.Item

	n, err := storage.FetchNode(p.host, node)
	if err != nil {
		log.Error(err)
		return nil, xmpp.ErrInternalServerError
	}
	if n == nil {
		// does not exist
		return nil, xmpp.ErrItemNotFound
	}
	nodeItems, err := storage.FetchNodeItems(p.host, node)
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

func (p *discoInfoProvider) Identities(_, _ *jid.JID, _ string) []xep0030.Identity { return nil }

func (p *discoInfoProvider) Features(_, _ *jid.JID, _ string) ([]xep0030.Feature, *xmpp.StanzaError) {
	return nil, nil
}

func (p *discoInfoProvider) Form(_, _ *jid.JID, _ string) (*xep0004.DataForm, *xmpp.StanzaError) {
	return nil, nil
}

func (p *discoInfoProvider) isSubscribedTo(contact *jid.JID, userJID *jid.JID) bool {
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
