/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"context"
	"sync"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/util/runqueue"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	discoInfoNamespace  = "http://jabber.org/protocol/disco#info"
	discoItemsNamespace = "http://jabber.org/protocol/disco#items"
)

// DiscoInfo represents a disco info server stream module.
type DiscoInfo struct {
	mu          sync.RWMutex
	router      *router.Router
	srvProvider *serverProvider
	providers   map[string]InfoProvider
	runQueue    *runqueue.RunQueue
}

// New returns a disco info IQ handler module.
func New(router *router.Router) *DiscoInfo {
	di := &DiscoInfo{
		router:      router,
		srvProvider: &serverProvider{router: router},
		providers:   make(map[string]InfoProvider),
		runQueue:    runqueue.New("xep0030"),
	}
	di.RegisterServerFeature(discoItemsNamespace)
	di.RegisterServerFeature(discoInfoNamespace)
	di.RegisterAccountFeature(discoItemsNamespace)
	di.RegisterAccountFeature(discoInfoNamespace)
	return di
}

// RegisterServerItem registers a new item associated to server domain.
func (x *DiscoInfo) RegisterServerItem(item Item) {
	x.srvProvider.registerServerItem(item)
}

// UnregisterServerItem unregisters a previously registered server item.
func (x *DiscoInfo) UnregisterServerItem(item Item) {
	x.srvProvider.unregisterServerItem(item)
}

// RegisterServerFeature registers a new feature associated to server domain.
func (x *DiscoInfo) RegisterServerFeature(feature string) {
	x.srvProvider.registerServerFeature(feature)
}

// UnregisterServerFeature unregisters a previously registered server feature.
func (x *DiscoInfo) UnregisterServerFeature(feature string) {
	x.srvProvider.unregisterServerFeature(feature)
}

// RegisterAccountFeature registers a new feature associated to all account domains.
func (x *DiscoInfo) RegisterAccountFeature(feature string) {
	x.srvProvider.registerAccountFeature(feature)
}

// UnregisterAccountFeature unregisters a previously registered account feature.
func (x *DiscoInfo) UnregisterAccountFeature(feature string) {
	x.srvProvider.unregisterAccountFeature(feature)
}

// RegisterProvider registers a new disco info provider associated to a domain.
func (x *DiscoInfo) RegisterProvider(domain string, provider InfoProvider) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.providers[domain] = provider
}

// UnregisterProvider unregisters a previously registered disco info provider.
func (x *DiscoInfo) UnregisterProvider(domain string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	delete(x.providers, domain)
}

// MatchesIQ returns whether or not an IQ should be
// processed by the disco info module.
func (x *DiscoInfo) MatchesIQ(iq *xmpp.IQ) bool {
	q := iq.Elements().Child("query")
	if q == nil {
		return false
	}
	return iq.IsGet() && (q.Namespace() == discoInfoNamespace || q.Namespace() == discoItemsNamespace)
}

// ProcessIQ processes a disco info IQ taking according actions over the associated stream.
func (x *DiscoInfo) ProcessIQ(ctx context.Context, iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		x.processIQ(ctx, iq)
	})
}

// Shutdown shuts down disco info module.
func (x *DiscoInfo) Shutdown() error {
	c := make(chan struct{})
	x.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}

func (x *DiscoInfo) processIQ(ctx context.Context, iq *xmpp.IQ) {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()

	var prov InfoProvider
	if x.router.IsLocalHost(toJID.Domain()) {
		if p := x.providers[toJID.String()]; p != nil {
			prov = p
		} else {
			prov = x.srvProvider
		}
	} else {
		prov = x.providers[toJID.Domain()]
		if prov == nil {
			_ = x.router.Route(ctx, iq.ItemNotFoundError())
			return
		}
	}
	q := iq.Elements().Child("query")
	node := q.Attributes().Get("node")
	if q != nil {
		switch q.Namespace() {
		case discoInfoNamespace:
			x.sendDiscoInfo(ctx, prov, toJID, fromJID, node, iq)
			return
		case discoItemsNamespace:
			x.sendDiscoItems(ctx, prov, toJID, fromJID, node, iq)
			return
		}
	}
	_ = x.router.Route(ctx, iq.BadRequestError())
}

func (x *DiscoInfo) sendDiscoInfo(ctx context.Context, prov InfoProvider, toJID, fromJID *jid.JID, node string, iq *xmpp.IQ) {
	features, sErr := prov.Features(ctx, toJID, fromJID, node)
	if sErr != nil {
		_ = x.router.Route(ctx, xmpp.NewErrorStanzaFromStanza(iq, sErr, nil))
		return
	} else if len(features) == 0 {
		_ = x.router.Route(ctx, iq.ItemNotFoundError())
		return
	}
	result := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", discoInfoNamespace)

	identities := prov.Identities(ctx, toJID, fromJID, node)
	for _, identity := range identities {
		identityEl := xmpp.NewElementName("identity")
		identityEl.SetAttribute("category", identity.Category)
		if len(identity.Type) > 0 {
			identityEl.SetAttribute("type", identity.Type)
		}
		if len(identity.Name) > 0 {
			identityEl.SetAttribute("name", identity.Name)
		}
		query.AppendElement(identityEl)
	}
	for _, feature := range features {
		featureEl := xmpp.NewElementName("feature")
		featureEl.SetAttribute("var", feature)
		query.AppendElement(featureEl)
	}
	form, sErr := prov.Form(ctx, toJID, fromJID, node)
	if sErr != nil {
		_ = x.router.Route(ctx, xmpp.NewErrorStanzaFromStanza(iq, sErr, nil))
		return
	}
	if form != nil {
		query.AppendElement(form.Element())
	}
	result.AppendElement(query)
	_ = x.router.Route(ctx, result)
}

func (x *DiscoInfo) sendDiscoItems(ctx context.Context, prov InfoProvider, toJID, fromJID *jid.JID, node string, iq *xmpp.IQ) {
	items, sErr := prov.Items(ctx, toJID, fromJID, node)
	if sErr != nil {
		_ = x.router.Route(ctx, xmpp.NewErrorStanzaFromStanza(iq, sErr, nil))
		return
	}
	result := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", discoItemsNamespace)
	for _, item := range items {
		itemEl := xmpp.NewElementName("item")
		itemEl.SetAttribute("jid", item.Jid)
		if len(item.Name) > 0 {
			itemEl.SetAttribute("name", item.Name)
		}
		if len(item.Node) > 0 {
			itemEl.SetAttribute("node", item.Node)
		}
		query.AppendElement(itemEl)
	}
	result.AppendElement(query)
	_ = x.router.Route(ctx, result)
}
