/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"crypto/sha256"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/log"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/runqueue"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// <feature var='http://jabber.org/protocol/pubsub#access-presence'/>          [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#auto-create'/>              [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#auto-subscribe'/>           [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#config-node'/>              [DONE]
// <feature var='http://jabber.org/protocol/pubsub#create-and-configure'/>     [DONE]
// <feature var='http://jabber.org/protocol/pubsub#create-nodes'/>             [DONE]
// <feature var='http://jabber.org/protocol/pubsub#filtered-notifications'/>   [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#persistent-items'/>         [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#publish'/>                  [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#retrieve-items'/>           [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#subscribe'/>                [PENDING]

const (
	pubSubNamespace      = "http://jabber.org/protocol/pubsub"
	pubSubOwnerNamespace = "http://jabber.org/protocol/pubsub#owner"
	pubSubEventNamespace = "http://jabber.org/protocol/pubsub#event"

	pubSubErrorNamespace = "http://jabber.org/protocol/pubsub#errors"
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
	DeliverNotifications:  true,
	DeliverPayloads:       true,
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
	if pubSub == nil {
		return false
	}
	switch pubSub.Namespace() {
	case pubSubNamespace, pubSubOwnerNamespace:
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
	case pubSubNamespace:
		x.processRequest(iq, pubSub)
	case pubSubOwnerNamespace:
		x.processOwnerRequest(iq, pubSub)
	}
}

func (x *Pep) processRequest(iq *xmpp.IQ, pubSub xmpp.XElement) {
	// Create node
	// https://xmpp.org/extensions/xep-0060.html#owner-create
	if createCmd := pubSub.Elements().Child("create"); createCmd != nil && iq.IsSet() {
		x.withNode(iq, pubSub, createCmd, true, x.createNode)
		return
	}

	_ = x.router.Route(iq.FeatureNotImplementedError())
}

func (x *Pep) processOwnerRequest(iq *xmpp.IQ, pubSub xmpp.XElement) {
	// Configure node
	// https://xmpp.org/extensions/xep-0060.html#owner-configure
	if configureCmd := pubSub.Elements().Child("configure"); configureCmd != nil {
		if iq.IsGet() {
			// send configuration form
			x.withNode(iq, pubSub, configureCmd, true, x.sendConfigurationForm)
		} else if iq.IsSet() {
			// update node configuration
			x.withNode(iq, pubSub, configureCmd, true, x.configureNode)
		} else {
			_ = x.router.Route(iq.ServiceUnavailableError())
		}
		return
	}

	// Manage affiliations
	// https://xmpp.org/extensions/xep-0060.html#owner-affiliations
	if affiliationsCmd := pubSub.Elements().Child("affiliations"); affiliationsCmd != nil {
		if iq.IsGet() {
			x.withNode(iq, pubSub, affiliationsCmd, true, x.retrieveAffiliations)
		} else if iq.IsSet() {
			x.withNode(iq, pubSub, affiliationsCmd, true, x.updateAffiliations)
		} else {
			_ = x.router.Route(iq.ServiceUnavailableError())
		}
		return
	}

	// Delete node
	// https://xmpp.org/extensions/xep-0060.html#owner-delete
	if deleteCmd := pubSub.Elements().Child("delete"); deleteCmd != nil && iq.IsSet() {
		x.withNode(iq, pubSub, deleteCmd, true, x.deleteNode)
		return
	}

	_ = x.router.Route(iq.FeatureNotImplementedError())
}

func (x *Pep) createNode(iq *xmpp.IQ, pubSubEl, _ xmpp.XElement, node *pubsubmodel.Node, host, nodeID string) {
	if node != nil {
		_ = x.router.Route(iq.ConflictError())
		return
	}
	node = &pubsubmodel.Node{
		Host: host,
		Name: nodeID,
	}
	if configEl := pubSubEl.Elements().Child("configure"); configEl != nil {
		form, err := xep0004.NewFormFromElement(configEl)
		if err != nil {
			_ = x.router.Route(iq.BadRequestError())
			return
		}
		opts, err := pubsubmodel.NewOptionsFromSubmitForm(form)
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
	log.Infof("pep: created node (host: %s, node_id: %s)", host, nodeID)

	// create owner affiliation
	ownerAffiliation := &pubsubmodel.Affiliation{
		JID:         host,
		Affiliation: pubsubmodel.Owner,
	}
	if err := storage.UpsertPubSubNodeAffiliation(ownerAffiliation, host, nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// create owner subscription
	ownerSub := &pubsubmodel.Subscription{
		SubID:        subscriptionID(host, pubsubmodel.Subscribed, host, nodeID),
		JID:          host,
		Subscription: pubsubmodel.Subscribed,
	}
	if err := storage.UpsertPubSubNodeSubscription(ownerSub, host, nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}

	// reply
	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) sendConfigurationForm(iq *xmpp.IQ, _, cmdElem xmpp.XElement, node *pubsubmodel.Node, host, nodeID string) {
	if node == nil {
		_ = x.router.Route(iq.ItemNotFoundError())
		return
	}
	// compose config form response
	configureNode := xmpp.NewElementName("configure")
	configureNode.SetAttribute("node", nodeID)
	configureNode.AppendElement(node.Options.Form().Element())

	pubSubNode := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	pubSubNode.AppendElement(configureNode)

	res := iq.ResultIQ()
	res.AppendElement(pubSubNode)

	log.Infof("pep: sent configuration form (host: %s, node_id: %s)", host, nodeID)

	// reply
	_ = x.router.Route(res)
}

func (x *Pep) configureNode(iq *xmpp.IQ, _, cmdElem xmpp.XElement, node *pubsubmodel.Node, host, nodeID string) {
	if node == nil {
		_ = x.router.Route(iq.ItemNotFoundError())
		return
	}
	formEl := cmdElem.Elements().ChildNamespace("x", xep0004.FormNamespace)
	if formEl == nil {
		_ = x.router.Route(iq.NotAcceptableError())
		return
	}
	configForm, err := xep0004.NewFormFromElement(formEl)
	if err != nil {
		_ = x.router.Route(iq.NotAcceptableError())
		return
	}
	nodeOpts, err := pubsubmodel.NewOptionsFromSubmitForm(configForm)
	if err != nil {
		_ = x.router.Route(iq.NotAcceptableError())
		return
	}
	node.Options = *nodeOpts

	// update node
	if err := storage.UpsertPubSubNode(node); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// notify config update
	if node.Options.DeliverNotifications && node.Options.NotifyConfig {
		configElem := xmpp.NewElementName("configuration")
		configElem.SetAttribute("node", nodeID)

		if node.Options.DeliverPayloads {
			configElem.AppendElement(node.Options.ResultForm().Element())
		}
		// fetch subscriptions
		subscriptions, err := storage.FetchPubSubNodeSubscriptions(host, nodeID)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
		x.notify(configElem, subscriptions, host)
	}
	log.Infof("pep: node configuration updated (host: %s, node_id: %s)", host, nodeID)

	// reply
	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) deleteNode(iq *xmpp.IQ, _, _ xmpp.XElement, node *pubsubmodel.Node, host, nodeID string) {
	if node == nil {
		_ = x.router.Route(iq.ItemNotFoundError())
		return
	}
	// fetch subscriptions
	subscriptions, err := storage.FetchPubSubNodeSubscriptions(host, nodeID)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// delete node
	if err := storage.DeletePubSubNode(host, nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// notify delete
	if node.Options.DeliverNotifications && node.Options.NotifyDelete {
		deleteElem := xmpp.NewElementName("delete")
		deleteElem.SetAttribute("node", nodeID)

		x.notify(deleteElem, subscriptions, host)
	}
	log.Infof("pep: deleted node (host: %s, node_id: %s)", host, nodeID)

	// reply
	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) retrieveAffiliations(iq *xmpp.IQ, _, _ xmpp.XElement, node *pubsubmodel.Node, host, nodeID string) {
	if node == nil {
		_ = x.router.Route(iq.ItemNotFoundError())
		return
	}
	// fetch affiliations
	affiliations, err := storage.FetchPubSubNodeAffiliations(host, nodeID)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// compose response
	affiliationsElem := xmpp.NewElementName("affiliations")
	affiliationsElem.SetAttribute("node", nodeID)

	for _, aff := range affiliations {
		affElem := xmpp.NewElementName("affiliation")
		affElem.SetAttribute("jid", aff.JID)
		affElem.SetAttribute("affiliation", aff.Affiliation)
	}
	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	pubSubElem.AppendElement(affiliationsElem)
	iqRes.AppendElement(pubSubElem)

	log.Infof("pep: retrieved affiliations (host: %s, node_id: %s)", host, nodeID)

	// reply
	_ = x.router.Route(iqRes)
}

func (x *Pep) updateAffiliations(iq *xmpp.IQ, _, cmdElem xmpp.XElement, node *pubsubmodel.Node, host, nodeID string) {
	if node == nil {
		_ = x.router.Route(iq.ItemNotFoundError())
		return
	}
	// update affiliations
	for _, affElem := range cmdElem.Elements().Children("affiliation") {
		var aff pubsubmodel.Affiliation
		aff.JID = affElem.Attributes().Get("jid")
		aff.Affiliation = affElem.Attributes().Get("affiliation")

		if aff.JID == iq.FromJID().ToBareJID().String() && aff.Affiliation == pubsubmodel.None {
			// ignore node owner resign
			continue
		}
		var err error
		switch aff.Affiliation {
		case pubsubmodel.Owner, pubsubmodel.Member, pubsubmodel.Publishers:
			err = storage.UpsertPubSubNodeAffiliation(&aff, host, nodeID)
		case pubsubmodel.None:
			err = storage.DeletePubSubNodeAffiliation(aff.JID, host, nodeID)
		default:
			_ = x.router.Route(iq.BadRequestError())
			return
		}
		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
	}
	log.Infof("pep: modified affiliations (host: %s, node_id: %s)", host, nodeID)

	// reply
	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) notify(notificationElem xmpp.XElement, subscriptions []pubsubmodel.Subscription, host string) {
	hostJID, _ := jid.NewWithString(host, true)
	for _, subscription := range subscriptions {
		if subscription.Subscription != pubsubmodel.Subscribed {
			continue
		}
		toJID, _ := jid.NewWithString(subscription.JID, true)

		msg := xmpp.NewMessageType(uuid.New().String(), xmpp.HeadlineType)
		msg.SetFromJID(hostJID)
		msg.SetToJID(toJID)
		eventElem := xmpp.NewElementNamespace("event", pubSubEventNamespace)
		eventElem.AppendElement(notificationElem)
		msg.AppendElement(eventElem)

		_ = x.router.Route(msg)
	}
}

func (x *Pep) withNode(iq *xmpp.IQ, pubSubEl, cmdElem xmpp.XElement, asOwner bool, fn func(iq *xmpp.IQ, pubSubEl, cmdElem xmpp.XElement, node *pubsubmodel.Node, host, nodeID string)) {
	if asOwner && iq.FromJID().ToBareJID().String() != iq.ToJID().ToBareJID().String() {
		_ = x.router.Route(iq.ForbiddenError())
		return
	}
	nodeName := cmdElem.Attributes().Get("node")
	if len(nodeName) == 0 {
		_ = x.router.Route(nodeIDRequiredError(iq))
		return
	}
	host := iq.FromJID().ToBareJID().String()

	// fetch node from storage
	node, err := storage.FetchPubSubNode(host, nodeName)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	fn(iq, pubSubEl, cmdElem, node, host, nodeName)
	return
}

func nodeIDRequiredError(stanza xmpp.Stanza) xmpp.Stanza {
	errorElements := []xmpp.XElement{xmpp.NewElementNamespace("nodeid-required", pubSubErrorNamespace)}
	return xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrNotAcceptable, errorElements)
}

func subscriptionID(jid, subscription, host, name string) string {
	h := sha256.New()
	h.Write([]byte(jid + subscription + host + name))
	return string(h.Sum(nil))
}
