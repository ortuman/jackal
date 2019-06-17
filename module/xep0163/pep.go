/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"github.com/ortuman/jackal/log"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/runqueue"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
)

const pepNamespace = "http://jabber.org/protocol/pubsub"

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

var defaultNodeOptions = []pubsubmodel.Option{
	{},
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
	return iq.Elements().ChildNamespace("pubsub", pepNamespace) != nil
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
	pubSub := iq.Elements().ChildNamespace("pubsub", pepNamespace)

	if createNode := pubSub.Elements().Child("create"); createNode != nil {
		nodeCfg := pubSub.Elements().Child("configure")

		x.createNode(iq, createNode, nodeCfg)
	}
}

func (x *Pep) createNode(iq *xmpp.IQ, nodeEl xmpp.XElement, configEl xmpp.XElement) {
	nodeName := nodeEl.Attributes().Get("node")
	if len(nodeName) == 0 {
		_ = x.router.Route(iq.BadRequestError())
		return
	}
	node := &pubsubmodel.Node{
		Host: iq.FromJID().ToBareJID().String(),
		Name: nodeName,
	}
	if configEl != nil {
		// TODO(ortuman): attach node options
	} else {
		// apply default configuration
		node.Options = defaultNodeOptions
	}
	if err := storage.InsertOrUpdatePubSubNode(node); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}

	_ = x.router.Route(iq.ResultIQ())
}
