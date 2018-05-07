/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"sort"

	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
)

const (
	discoInfoNamespace  = "http://jabber.org/protocol/disco#info"
	discoItemsNamespace = "http://jabber.org/protocol/disco#items"
)

// DiscoFeature represents a disco info feature entity.
type DiscoFeature = string

// DiscoItem represents a disco info item entity.
type DiscoItem struct {
	Jid  string
	Name string
	Node string
}

// DiscoIdentity represents a disco info identity entity.
type DiscoIdentity struct {
	Category string
	Type     string
	Name     string
}

// XEPDiscoInfo represents a disco info server stream module.
type XEPDiscoInfo struct {
	stm        c2s.Stream
	identities []DiscoIdentity
	features   []DiscoFeature
	items      []DiscoItem
}

// New returns a disco info IQ handler module.
func New(stm c2s.Stream) *XEPDiscoInfo {
	return &XEPDiscoInfo{stm: stm}
}

// Identities returns disco info module's identities.
func (x *XEPDiscoInfo) Identities() []DiscoIdentity {
	return x.identities
}

// SetIdentities sets disco info module's identities.
func (x *XEPDiscoInfo) SetIdentities(identities []DiscoIdentity) {
	x.identities = identities
}

// Features returns disco info module's features.
func (x *XEPDiscoInfo) Features() []DiscoFeature {
	return x.features
}

// SetFeatures sets disco info module's features.
func (x *XEPDiscoInfo) SetFeatures(features []DiscoFeature) {
	x.features = features
}

// Items returns disco info module's items.
func (x *XEPDiscoInfo) Items() []DiscoItem {
	return x.items
}

// SetItems sets disco info module's items.
func (x *XEPDiscoInfo) SetItems(items []DiscoItem) {
	x.items = items
}

// AssociatedNamespaces returns namespaces associated
// with disco info module.
func (x *XEPDiscoInfo) AssociatedNamespaces() []string {
	return []string{discoInfoNamespace, discoItemsNamespace}
}

// Done signals stream termination.
func (x *XEPDiscoInfo) Done() {
}

// MatchesIQ returns whether or not an IQ should be
// processed by the disco info module.
func (x *XEPDiscoInfo) MatchesIQ(iq *xml.IQ) bool {
	q := iq.Elements().Child("query")
	if q == nil {
		return false
	}
	return iq.IsGet() && (q.Namespace() == discoInfoNamespace || q.Namespace() == discoItemsNamespace)
}

// ProcessIQ processes a disco info IQ taking according actions
// over the associated stream.
func (x *XEPDiscoInfo) ProcessIQ(iq *xml.IQ) {
	if !iq.ToJID().IsServer() {
		x.stm.SendElement(iq.FeatureNotImplementedError())
		return
	}
	q := iq.Elements().Child("query")
	switch q.Namespace() {
	case discoInfoNamespace:
		x.sendDiscoInfo(iq)
	case discoItemsNamespace:
		x.sendDiscoItems(iq)
	}
}

func (x *XEPDiscoInfo) sendDiscoInfo(iq *xml.IQ) {
	sort.Slice(x.features, func(i, j int) bool { return x.features[i] < x.features[j] })

	result := iq.ResultIQ()
	query := xml.NewElementNamespace("query", discoInfoNamespace)

	for _, identity := range x.identities {
		identityEl := xml.NewElementName("identity")
		identityEl.SetAttribute("category", identity.Category)
		if len(identity.Type) > 0 {
			identityEl.SetAttribute("type", identity.Type)
		}
		if len(identity.Name) > 0 {
			identityEl.SetAttribute("name", identity.Name)
		}
		query.AppendElement(identityEl)
	}
	for _, feature := range x.features {
		featureEl := xml.NewElementName("feature")
		featureEl.SetAttribute("var", feature)
		query.AppendElement(featureEl)
	}

	result.AppendElement(query)
	x.stm.SendElement(result)
}

func (x *XEPDiscoInfo) sendDiscoItems(iq *xml.IQ) {
	result := iq.ResultIQ()
	query := xml.NewElementNamespace("query", discoItemsNamespace)

	for _, item := range x.items {
		itemEl := xml.NewElementName("item")
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
	x.stm.SendElement(result)
}
