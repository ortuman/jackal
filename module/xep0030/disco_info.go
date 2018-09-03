/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"sync"

	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const mailboxSize = 2048

const (
	discoInfoNamespace  = "http://jabber.org/protocol/disco#info"
	discoItemsNamespace = "http://jabber.org/protocol/disco#items"
)

// Feature represents a disco info feature entity.
type Feature = string

// Identity represents a disco info identity entity.
type Identity struct {
	Category string
	Type     string
	Name     string
}

// Item represents a disco info item entity.
type Item struct {
	Jid  string
	Name string
	Node string
}

// Provider represents a generic disco info domain provider.
type Provider interface {
	// Identities returns all identities associated to the provider.
	Identities(toJID, fromJID *jid.JID, node string) []Identity

	// Items returns all items associated to the provider.
	// A proper stanza error should be returned in case an error occurs.
	Items(toJID, fromJID *jid.JID, node string) ([]Item, *xmpp.StanzaError)

	// Features returns all features associated to the provider.
	// A proper stanza error should be returned in case an error occurs.
	Features(toJID, fromJID *jid.JID, node string) ([]Feature, *xmpp.StanzaError)
}

// DiscoInfo represents a disco info server stream module.
type DiscoInfo struct {
	mu          sync.RWMutex
	actorCh     chan func()
	shutdownCh  <-chan struct{}
	srvProvider *serverProvider
	providers   map[string]Provider
}

// New returns a disco info IQ handler module.
func New(shutdownCh <-chan struct{}) *DiscoInfo {
	di := &DiscoInfo{
		srvProvider: &serverProvider{},
		providers:   make(map[string]Provider),
		actorCh:     make(chan func(), mailboxSize),
		shutdownCh:  shutdownCh,
	}
	go di.loop()
	di.RegisterServerFeature(discoItemsNamespace)
	di.RegisterServerFeature(discoInfoNamespace)
	di.RegisterAccountFeature(discoItemsNamespace)
	di.RegisterAccountFeature(discoInfoNamespace)
	return di
}

// RegisterServerFeature registers a new feature associated to server domain.
func (di *DiscoInfo) RegisterServerFeature(feature string) {
	di.srvProvider.registerServerFeature(feature)
}

// UnregisterServerFeature unregisters a previous registered server feature.
func (di *DiscoInfo) UnregisterServerFeature(feature string) {
	di.srvProvider.unregisterServerFeature(feature)
}

// RegisterAccountFeature registers a new feature associated to all account domains.
func (di *DiscoInfo) RegisterAccountFeature(feature string) {
	di.srvProvider.registerAccountFeature(feature)
}

// UnregisterAccountFeature unregisters a previous registered account feature.
func (di *DiscoInfo) UnregisterAccountFeature(feature string) {
	di.srvProvider.unregisterAccountFeature(feature)
}

// RegisterProvider registers a new disco info provider given a domain name.
func (di *DiscoInfo) RegisterProvider(domain string, provider Provider) {
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
func (di *DiscoInfo) ProcessIQ(iq *xmpp.IQ, stm stream.C2S) {
	di.actorCh <- func() { di.processIQ(iq, stm) }
}

// runs on it's own goroutine
func (di *DiscoInfo) loop() {
	for {
		select {
		case f := <-di.actorCh:
			f()
		case <-di.shutdownCh:
			return
		}
	}
}

func (di *DiscoInfo) processIQ(iq *xmpp.IQ, stm stream.C2S) {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()

	var prov Provider
	if host.IsLocalHost(toJID.Domain()) {
		prov = di.srvProvider
	} else {
		prov = di.providers[toJID.Domain()]
		if prov == nil {
			stm.SendElement(iq.ItemNotFoundError())
			return
		}
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

func (di *DiscoInfo) sendDiscoInfo(prov Provider, toJID, fromJID *jid.JID, node string, iq *xmpp.IQ, stm stream.C2S) {
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
	result.AppendElement(query)
	stm.SendElement(result)
}

func (di *DiscoInfo) sendDiscoItems(prov Provider, toJID, fromJID *jid.JID, node string, iq *xmpp.IQ, stm stream.C2S) {
	items, sErr := prov.Items(toJID, fromJID, node)
	if sErr != nil {
		stm.SendElement(xmpp.NewErrorStanzaFromStanza(iq, sErr, nil))
		return
	} else if len(items) == 0 {
		stm.SendElement(iq.ItemNotFoundError())
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
