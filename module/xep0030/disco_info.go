/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"sync"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const mailboxSize = 2048

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
	actorCh     chan func()
	shutdownCh  chan chan error
}

// New returns a disco info IQ handler module.
func New(router *router.Router) *DiscoInfo {
	di := &DiscoInfo{
		router:      router,
		srvProvider: &serverProvider{router: router},
		providers:   make(map[string]InfoProvider),
		actorCh:     make(chan func(), mailboxSize),
		shutdownCh:  make(chan chan error),
	}
	go di.loop()
	di.RegisterServerFeature(discoItemsNamespace)
	di.RegisterServerFeature(discoInfoNamespace)
	di.RegisterAccountFeature(discoItemsNamespace)
	di.RegisterAccountFeature(discoInfoNamespace)
	return di
}

// RegisterServerItem registers a new item associated to server domain.
func (di *DiscoInfo) RegisterServerItem(item Item) {
	di.srvProvider.registerServerItem(item)
}

// UnregisterServerItem unregisters a previously registered server item.
func (di *DiscoInfo) UnregisterServerItem(item Item) {
	di.srvProvider.unregisterServerItem(item)
}

// RegisterServerFeature registers a new feature associated to server domain.
func (di *DiscoInfo) RegisterServerFeature(feature string) {
	di.srvProvider.registerServerFeature(feature)
}

// UnregisterServerFeature unregisters a previously registered server feature.
func (di *DiscoInfo) UnregisterServerFeature(feature string) {
	di.srvProvider.unregisterServerFeature(feature)
}

// RegisterAccountFeature registers a new feature associated to all account domains.
func (di *DiscoInfo) RegisterAccountFeature(feature string) {
	di.srvProvider.registerAccountFeature(feature)
}

// UnregisterAccountFeature unregisters a previously registered account feature.
func (di *DiscoInfo) UnregisterAccountFeature(feature string) {
	di.srvProvider.unregisterAccountFeature(feature)
}

// RegisterProvider registers a new disco info provider associated to a domain.
func (di *DiscoInfo) RegisterProvider(domain string, provider InfoProvider) {
	di.mu.Lock()
	defer di.mu.Unlock()
	di.providers[domain] = provider
}

// UnregisterProvider unregisters a previously registered disco info provider.
func (di *DiscoInfo) UnregisterProvider(domain string) {
	di.mu.Lock()
	defer di.mu.Unlock()
	delete(di.providers, domain)
}

// MatchesIQ returns whether or not an IQ should be
// processed by the disco info module.
func (di *DiscoInfo) MatchesIQ(iq *xmpp.IQ) bool {
	q := iq.Elements().Child("query")
	if q == nil {
		return false
	}
	return iq.IsGet() && (q.Namespace() == discoInfoNamespace || q.Namespace() == discoItemsNamespace)
}

// ProcessIQ processes a disco info IQ taking according actions
// over the associated stream.
func (di *DiscoInfo) ProcessIQ(iq *xmpp.IQ, stm stream.Stream) {
	di.actorCh <- func() { di.processIQ(iq, stm) }
}

// Shutdown shuts down disco info module.
func (di *DiscoInfo) Shutdown() error {
	c := make(chan error)
	di.shutdownCh <- c
	return <-c
}

// runs on it's own goroutine
func (di *DiscoInfo) loop() {
	for {
		select {
		case f := <-di.actorCh:
			f()
		case c := <-di.shutdownCh:
			c <- nil
			return
		}
	}
}

func (di *DiscoInfo) processIQ(iq *xmpp.IQ, stm stream.Stream) {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()

	var prov InfoProvider
	if di.router.IsLocalHost(toJID.Domain()) {
		prov = di.srvProvider
	} else {
		prov = di.providers[toJID.Domain()]
		if prov == nil {
			stm.SendElement(iq.ItemNotFoundError())
			return
		}
	}
	if prov == nil {
		stm.SendElement(iq.ItemNotFoundError())
		return
	}
	q := iq.Elements().Child("query")
	node := q.Attributes().Get("node")
	if q != nil {
		switch q.Namespace() {
		case discoInfoNamespace:
			di.sendDiscoInfo(prov, toJID, fromJID, node, iq, stm)
			return
		case discoItemsNamespace:
			di.sendDiscoItems(prov, toJID, fromJID, node, iq, stm)
			return
		}
	}
	stm.SendElement(iq.BadRequestError())
}

func (di *DiscoInfo) sendDiscoInfo(prov InfoProvider, toJID, fromJID *jid.JID, node string, iq *xmpp.IQ, stm stream.Stream) {
	features, sErr := prov.Features(toJID, fromJID, node)
	if sErr != nil {
		stm.SendElement(xmpp.NewErrorStanzaFromStanza(iq, sErr, nil))
		return
	} else if len(features) == 0 {
		stm.SendElement(iq.ItemNotFoundError())
		return
	}
	result := iq.ResultIQ()
	query := xmpp.NewElementNamespace("query", discoInfoNamespace)

	identities := prov.Identities(toJID, fromJID, node)
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
	form, sErr := prov.Form(toJID, fromJID, node)
	if sErr != nil {
		stm.SendElement(xmpp.NewErrorStanzaFromStanza(iq, sErr, nil))
		return
	}
	if form != nil {
		query.AppendElement(form.Element())
	}
	result.AppendElement(query)
	stm.SendElement(result)
}

func (di *DiscoInfo) sendDiscoItems(prov InfoProvider, toJID, fromJID *jid.JID, node string, iq *xmpp.IQ, stm stream.Stream) {
	items, sErr := prov.Items(toJID, fromJID, node)
	if sErr != nil {
		stm.SendElement(xmpp.NewErrorStanzaFromStanza(iq, sErr, nil))
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
	stm.SendElement(result)
}
