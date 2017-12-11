/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"sort"

	"github.com/ortuman/jackal/concurrent"
	"github.com/ortuman/jackal/xml"
)

const (
	discoInfoNamespace  = "http://jabber.org/protocol/disco#info"
	discoItemsNamespace = "http://jabber.org/protocol/disco#items"
)

type DiscoFeature string

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
	concurrent.DispatcherQueue
	strm       Stream
	identities []DiscoIdentity
	features   []DiscoFeature
	items      []DiscoItem
}

func NewXEPDiscoInfo(strm Stream) *XEPDiscoInfo {
	x := &XEPDiscoInfo{strm: strm}
	return x
}

func (x *XEPDiscoInfo) Identities() []DiscoIdentity {
	ch := make(chan []DiscoIdentity)
	x.Async(func() {
		ch <- x.identities
	})
	return <-ch
}

func (x *XEPDiscoInfo) SetIdentities(identities []DiscoIdentity) {
	x.Sync(func() {
		x.identities = identities
	})
}

func (x *XEPDiscoInfo) Features() []DiscoFeature {
	ch := make(chan []DiscoFeature)
	x.Async(func() {
		ch <- x.features
	})
	return <-ch
}

func (x *XEPDiscoInfo) SetFeatures(features []DiscoFeature) {
	x.Sync(func() {
		x.features = features
	})
}

func (x *XEPDiscoInfo) Items() []DiscoItem {
	ch := make(chan []DiscoItem)
	x.Async(func() {
		ch <- x.items
	})
	return <-ch
}

func (x *XEPDiscoInfo) SetItems(items []DiscoItem) {
	x.Sync(func() {
		x.items = items
	})
}

func (x *XEPDiscoInfo) MatchesIQ(iq *xml.IQ) bool {
	q := iq.FindElement("query")
	if q == nil {
		return false
	}
	return iq.IsGet() && (q.Namespace() == discoInfoNamespace || q.Namespace() == discoItemsNamespace)
}

func (x *XEPDiscoInfo) ProcessIQ(iq *xml.IQ) {
	x.Async(func() {
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
	if !iq.ToJID().IsServer() {
		x.strm.SendElement(iq.FeatureNotImplementedError())
		return
	}
	sort.Slice(x.features, func(i, j int) bool { return x.features[i] < x.features[j] })

	// TODO: Implement me!
}

func (x *XEPDiscoInfo) sendDiscoItems(iq *xml.IQ) {
	x.strm.SendElement(iq.FeatureNotImplementedError())
}
