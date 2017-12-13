/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"sort"

	"time"

	"sync"

	"github.com/ortuman/jackal/concurrent"
	"github.com/ortuman/jackal/xml"
)

const (
	discoInfoNamespace  = "http://jabber.org/protocol/disco#info"
	discoItemsNamespace = "http://jabber.org/protocol/disco#items"
)

type DiscoItem struct {
	Jid  string
	Name string
	Node string
}

type DiscoIdentity struct {
	Category string
	Type     string
	Name     string
}

type XEPDiscoInfo struct {
	sync.RWMutex
	queue      concurrent.OperationQueue
	strm       Stream
	identities []DiscoIdentity
	features   []string
	items      []DiscoItem
}

func NewXEPDiscoInfo(strm Stream) *XEPDiscoInfo {
	x := &XEPDiscoInfo{
		queue: concurrent.OperationQueue{
			QueueSize: 16,
			Timeout:   time.Second,
		},
		strm: strm,
	}
	return x
}

func (x *XEPDiscoInfo) Identities() []DiscoIdentity {
	x.RLock()
	defer x.RUnlock()
	return x.identities
}

func (x *XEPDiscoInfo) SetIdentities(identities []DiscoIdentity) {
	x.Lock()
	defer x.Unlock()
	x.identities = identities
}

func (x *XEPDiscoInfo) Features() []string {
	x.RLock()
	defer x.RUnlock()
	return x.features
}

func (x *XEPDiscoInfo) SetFeatures(features []string) {
	x.Lock()
	defer x.Unlock()
	x.features = features
}

func (x *XEPDiscoInfo) Items() []DiscoItem {
	x.RLock()
	defer x.RUnlock()
	return x.items
}

func (x *XEPDiscoInfo) SetItems(items []DiscoItem) {
	x.Lock()
	defer x.Unlock()
	x.items = items
}

func (x *XEPDiscoInfo) AssociatedNamespaces() []string {
	return []string{discoInfoNamespace, discoItemsNamespace}
}

func (x *XEPDiscoInfo) MatchesIQ(iq *xml.IQ) bool {
	q := iq.FindElement("query")
	if q == nil {
		return false
	}
	return iq.IsGet() && (q.Namespace() == discoInfoNamespace || q.Namespace() == discoItemsNamespace)
}

func (x *XEPDiscoInfo) ProcessIQ(iq *xml.IQ) {
	x.queue.Exec(func() {
		if !iq.ToJID().IsServer() {
			x.strm.SendElement(iq.FeatureNotImplementedError())
			return
		}
		q := iq.FindElement("query")
		switch q.Namespace() {
		case discoInfoNamespace:
			x.sendDiscoInfo(iq)
		case discoItemsNamespace:
			x.sendDiscoItems(iq)
		}
	})
}

func (x *XEPDiscoInfo) sendDiscoInfo(iq *xml.IQ) {
	sort.Slice(x.features, func(i, j int) bool { return x.features[i] < x.features[j] })

	result := iq.ResultIQ()
	query := xml.NewMutableElementNamespace("query", discoInfoNamespace)

	x.RLock()
	for _, identity := range x.identities {
		identityEl := xml.NewMutableElementName("identity")
		identityEl.SetAttribute("category", identity.Category)
		if len(identity.Type) > 0 {
			identityEl.SetAttribute("type", identity.Type)
		}
		if len(identity.Name) > 0 {
			identityEl.SetAttribute("name", identity.Name)
		}
		query.AppendElement(identityEl.Copy())
	}
	for _, feature := range x.features {
		featureEl := xml.NewMutableElementName("feature")
		featureEl.SetAttribute("var", feature)
		query.AppendElement(featureEl.Copy())
	}
	x.RUnlock()

	result.AppendElement(query.Copy())
	x.strm.SendElement(query)
}

func (x *XEPDiscoInfo) sendDiscoItems(iq *xml.IQ) {
	result := iq.ResultIQ()
	query := xml.NewMutableElementNamespace("query", discoItemsNamespace)

	x.RLock()
	for _, item := range x.items {
		itemEl := xml.NewMutableElementName("item")
		itemEl.SetAttribute("jid", item.Jid)
		if len(item.Name) > 0 {
			itemEl.SetAttribute("name", item.Name)
		}
		if len(item.Node) > 0 {
			itemEl.SetAttribute("node", item.Node)
		}
		query.AppendElement(itemEl.Copy())
	}
	x.RUnlock()

	result.AppendElement(query.Copy())
	x.strm.SendElement(query)
}
