/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"github.com/ortuman/jackal/log"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/runqueue"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
)

// <feature var='http://jabber.org/protocol/pubsub#access-presence'/>          [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#auto-create'/>              [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#auto-subscribe'/>           [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#config-node'/>              [PENDING] - Next
// <feature var='http://jabber.org/protocol/pubsub#create-and-configure'/>     [DONE]
// <feature var='http://jabber.org/protocol/pubsub#create-nodes'/>             [DONE]
// <feature var='http://jabber.org/protocol/pubsub#filtered-notifications'/>   [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#persistent-items'/>         [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#publish'/>                  [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#retrieve-items'/>           [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#subscribe'/>                [PENDING]

const (
	pepOwnerNamespace = "http://jabber.org/protocol/pubsub#owner"

	pepErrorNamespace = "http://jabber.org/protocol/pubsub#errors"

	pepNodeConfigNamespace = "http://jabber.org/protocol/pubsub#node_config"
)

var discoInfoFeatures = []string{
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

var defaultNodeOptions = pubsubmodel.Options{
	AccessModel:           pubsubmodel.Presence,
	PublishModel:          pubsubmodel.Publishers,
	SendLastPublishedItem: pubsubmodel.OnSubAndPresence,
	MaxItems:              1,
}

type Pep struct {
	router   *router.Router
	runQueue *runqueue.RunQueue
}

func New(disco *xep0030.DiscoInfo, router *router.Router) *Pep {
	p := &Pep{
		router:   router,
		runQueue: runqueue.New("xep0163"),
	}

	// register account identity and features
	if disco != nil {
		for _, feature := range discoInfoFeatures {
			disco.RegisterAccountFeature(feature)
		}
	}
	return p
}

// MatchesIQ returns whether or not an IQ should be processed by the PEP module.
func (x *Pep) MatchesIQ(iq *xmpp.IQ) bool {
	pubSub := iq.Elements().Child("pubsub")
	switch pubSub.Namespace() {
	case pepOwnerNamespace:
		return true
	}
	return false
}

// ProcessIQ processes a version IQ taking according actions over the associated stream.
func (x *Pep) ProcessIQ(iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		x.processIQ(iq)
	})
}

// Shutdown shuts down version module.
func (x *Pep) Shutdown() error {
	c := make(chan struct{})
	x.runQueue.Stop(func() { close(c) })
	<-c
	return nil
}

func (x *Pep) processIQ(iq *xmpp.IQ) {
	pubSub := iq.Elements().Child("pubsub")
	switch pubSub.Namespace() {
	case pepOwnerNamespace:
		x.processOwnerRequest(iq, pubSub)
		return
	}
}

func (x *Pep) processOwnerRequest(iq *xmpp.IQ, pubSub xmpp.XElement) {
	// Create node
	// https://xmpp.org/extensions/xep-0060.html#owner-create
	if createNode := pubSub.Elements().Child("create"); createNode != nil && iq.IsSet() {
		nodeCfg := pubSub.Elements().Child("configure")
		x.createNode(iq, createNode, nodeCfg)
		return
	}

	// Configure node
	// https://xmpp.org/extensions/xep-0060.html#owner-configure
	if configureNode := pubSub.Elements().Child("configure"); configureNode != nil {
		if iq.IsGet() {
			// send configuration form
			x.sendConfigurationForm(iq, configureNode)
		} else if iq.IsSet() {
			// update node configuration
			x.configureNode(iq, configureNode)
		}
		return
	}

	// Delete node
	// https://xmpp.org/extensions/xep-0060.html#owner-delete
	if deleteNode := pubSub.Elements().Child("delete"); deleteNode != nil && iq.IsSet() {
		x.deleteNode(iq, deleteNode)
		return
	}

	_ = x.router.Route(iq.FeatureNotImplementedError())
}

func (x *Pep) createNode(iq *xmpp.IQ, nodeEl xmpp.XElement, configEl xmpp.XElement) {
	if iq.FromJID().Node() != iq.ToJID().Node() {
		_ = x.router.Route(iq.ForbiddenError())
		return
	}
	nodeName := nodeEl.Attributes().Get("node")
	if len(nodeName) == 0 {
		_ = x.router.Route(nodeIDRequiredError(iq))
		return
	}
	host := iq.FromJID().ToBareJID().String()

	// check whether or not the node exists
	exists, err := storage.PubSubNodeExists(host, nodeName)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	if exists {
		_ = x.router.Route(iq.ConflictError())
		return
	}

	node := &pubsubmodel.Node{
		Host: host,
		Name: nodeName,
	}
	if configEl != nil {
		form, err := xep0004.NewFormFromElement(configEl)
		if err != nil {
			_ = x.router.Route(iq.BadRequestError())
			return
		}
		opts, err := pubsubmodel.NewOptionsFromForm(form)
		if err != nil {
			_ = x.router.Route(iq.BadRequestError())
			return
		}
		node.Options = *opts
	} else {
		// apply default configuration
		node.Options = defaultNodeOptions
	}

	// create node
	if err := storage.UpsertPubSubNode(node); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// create owner affiliation
	ownerAffiliation := &pubsubmodel.Affiliation{
		JID:         host,
		Affiliation: pubsubmodel.Owner,
	}
	if err := storage.UpsertPubSubNodeAffiliation(ownerAffiliation, host, nodeName); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}

	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) sendConfigurationForm(iq *xmpp.IQ, nodeEl xmpp.XElement) {
	if iq.FromJID().Node() != iq.ToJID().Node() {
		_ = x.router.Route(iq.ForbiddenError())
		return
	}
	nodeName := nodeEl.Attributes().Get("node")
	if len(nodeName) == 0 {
		_ = x.router.Route(nodeIDRequiredError(iq))
		return
	}
	host := iq.FromJID().ToBareJID().String()

	// fetch node configuration
	node, err := storage.FetchPubSubNode(host, nodeName)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	if node == nil {
		_ = x.router.Route(iq.ItemNotFoundError())
		return
	}
	// compose response
	configureNode := xmpp.NewElementName("configure")
	configureNode.SetAttribute("node", nodeName)
	configureNode.AppendElement(node.Options.Form().Element())

	pubSubNode := xmpp.NewElementNamespace("pubsub", pepOwnerNamespace)
	pubSubNode.AppendElement(configureNode)

	res := iq.ResultIQ()
	res.AppendElement(pubSubNode)

	// reply
	_ = x.router.Route(res)
}

func (x *Pep) configureNode(iq *xmpp.IQ, nodeEl xmpp.XElement) {
	if iq.FromJID().Node() != iq.ToJID().Node() {
		_ = x.router.Route(iq.ForbiddenError())
		return
	}
	nodeName := nodeEl.Attributes().Get("node")
	if len(nodeName) == 0 {
		_ = x.router.Route(iq.NotAcceptableError())
		return
	}
	//host := iq.FromJID().ToBareJID().String()
}

func (x *Pep) deleteNode(iq *xmpp.IQ, nodeEl xmpp.XElement) {
	if iq.FromJID().Node() != iq.ToJID().Node() {
		_ = x.router.Route(iq.ForbiddenError())
		return
	}
	nodeName := nodeEl.Attributes().Get("node")
	if len(nodeName) == 0 {
		_ = x.router.Route(nodeIDRequiredError(iq))
		return
	}
	host := iq.FromJID().ToBareJID().String()

	// check whether or not the node exists
	exists, err := storage.PubSubNodeExists(host, nodeName)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	if !exists {
		_ = x.router.Route(iq.ItemNotFoundError())
		return
	}
	// delete node
	if err := storage.DeletePubSubNode(host, nodeName); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}

	_ = x.router.Route(iq.ResultIQ())
}

func nodeIDRequiredError(stanza xmpp.Stanza) xmpp.Stanza {
	errorElements := []xmpp.XElement{xmpp.NewElementNamespace("nodeid-required", pepErrorNamespace)}
	return xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrNotAcceptable, errorElements)
}
