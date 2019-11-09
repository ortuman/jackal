/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"crypto/sha256"
	"fmt"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/runqueue"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// <feature var='http://jabber.org/protocol/pubsub#access-presence'/>          [DONE]
// <feature var='http://jabber.org/protocol/pubsub#auto-create'/>              [DONE]
// <feature var='http://jabber.org/protocol/pubsub#auto-subscribe'/>           [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#config-node'/>              [DONE]
// <feature var='http://jabber.org/protocol/pubsub#create-and-configure'/>     [DONE]
// <feature var='http://jabber.org/protocol/pubsub#create-nodes'/>             [DONE]
// <feature var='http://jabber.org/protocol/pubsub#filtered-notifications'/>   [PENDING]
// <feature var='http://jabber.org/protocol/pubsub#persistent-items'/>         [DONE]
// <feature var='http://jabber.org/protocol/pubsub#publish'/>                  [DONE]
// <feature var='http://jabber.org/protocol/pubsub#retrieve-items'/>           [DONE]
// <feature var='http://jabber.org/protocol/pubsub#subscribe'/>                [DONE]

// send last published item!!!

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
	PersistItems:          true,
	MaxItems:              1,
	SendLastPublishedItem: pubsubmodel.OnSubAndPresence,
	NotificationType:      xmpp.HeadlineType,
}

type commandOptions struct {
	allowedAffiliations  []string
	includeAffiliations  bool
	includeSubscriptions bool
	checkAccess          bool
	failOnNotFound       bool
}

type commandContext struct {
	host           string
	nodeID         string
	isAccountOwner bool
	node           *pubsubmodel.Node
	affiliations   []pubsubmodel.Affiliation
	subscriptions  []pubsubmodel.Subscription
	accessChecker  *accessChecker
}

type onlinePresenceProvider interface {
	OnlinePresencesMatchingJID(j *jid.JID) []model.OnlinePresence
}

type Pep struct {
	onlinePresenceProvider onlinePresenceProvider
	router                 *router.Router
	runQueue               *runqueue.RunQueue
}

func New(disco *xep0030.DiscoInfo, onlinePresenceProvider onlinePresenceProvider, router *router.Router) *Pep {
	p := &Pep{
		onlinePresenceProvider: onlinePresenceProvider,
		router:                 router,
		runQueue:               runqueue.New("xep0163"),
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

func (x *Pep) processRequest(iq *xmpp.IQ, pubSubEl xmpp.XElement) {
	// Create node
	if cmdEl := pubSubEl.Elements().Child("create"); cmdEl != nil && iq.IsSet() {
		x.withCommandContext(func(cmdCtx *commandContext) {
			x.create(cmdCtx, pubSubEl, iq)
		}, commandOptions{}, cmdEl, iq)
		return
	}
	// Publish
	if cmdEl := pubSubEl.Elements().Child("publish"); cmdEl != nil && iq.IsSet() {
		opts := commandOptions{
			allowedAffiliations:  []string{pubsubmodel.Owner, pubsubmodel.Member},
			includeSubscriptions: true,
		}
		x.withCommandContext(func(cmdCtx *commandContext) { x.publish(cmdCtx, cmdEl, iq) }, opts, cmdEl, iq)
		return
	}
	// Subscribe
	if cmdEl := pubSubEl.Elements().Child("subscribe"); cmdEl != nil && iq.IsSet() {
		opts := commandOptions{
			includeAffiliations: true,
			checkAccess:         true,
			failOnNotFound:      true,
		}
		x.withCommandContext(func(cmdCtx *commandContext) { x.subscribe(cmdCtx, cmdEl, iq) }, opts, cmdEl, iq)
		return
	}
	// Unsubscribe
	if cmdEl := pubSubEl.Elements().Child("unsubscribe"); cmdEl != nil && iq.IsSet() {
		opts := commandOptions{
			includeAffiliations:  true,
			includeSubscriptions: true,
			failOnNotFound:       true,
		}
		x.withCommandContext(func(cmdCtx *commandContext) {
			x.unsubscribe(cmdCtx, cmdEl, iq)
		}, opts, cmdEl, iq)
		return
	}
	// Retrieve items
	if cmdEl := pubSubEl.Elements().Child("items"); cmdEl != nil && iq.IsGet() {
		opts := commandOptions{
			includeSubscriptions: true,
			checkAccess:          true,
			failOnNotFound:       true,
		}
		x.withCommandContext(func(cmdCtx *commandContext) {
			x.retrieveItems(cmdCtx, cmdEl, iq)
		}, opts, cmdEl, iq)
		return
	}

	_ = x.router.Route(iq.ServiceUnavailableError())
}

func (x *Pep) processOwnerRequest(iq *xmpp.IQ, pubSub xmpp.XElement) {
	// Configure node
	if cmdEl := pubSub.Elements().Child("configure"); cmdEl != nil {
		if iq.IsGet() {
			// send configuration form
			opts := commandOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				failOnNotFound:      true,
			}
			x.withCommandContext(func(cmdCtx *commandContext) {
				x.sendConfigurationForm(cmdCtx, iq)
			}, opts, cmdEl, iq)
		} else if iq.IsSet() {
			// update node configuration
			opts := commandOptions{
				allowedAffiliations:  []string{pubsubmodel.Owner},
				includeSubscriptions: true,
				failOnNotFound:       true,
			}
			x.withCommandContext(func(cmdCtx *commandContext) { x.configure(cmdCtx, cmdEl, iq) }, opts, cmdEl, iq)
		} else {
			_ = x.router.Route(iq.ServiceUnavailableError())
		}
		return
	}
	// Manage affiliations
	if cmdEl := pubSub.Elements().Child("affiliations"); cmdEl != nil {
		if iq.IsGet() {
			opts := commandOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				includeAffiliations: true,
				failOnNotFound:      true,
			}
			x.withCommandContext(func(cmdCtx *commandContext) {
				x.retrieveAffiliations(cmdCtx, iq)
			}, opts, cmdEl, iq)
		} else if iq.IsSet() {
			opts := commandOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				failOnNotFound:      true,
			}
			x.withCommandContext(func(cmdCtx *commandContext) {
				x.updateAffiliations(cmdCtx, cmdEl, iq)
			}, opts, cmdEl, iq)
		} else {
			_ = x.router.Route(iq.ServiceUnavailableError())
		}
		return
	}
	// Manage subscriptions
	if cmdEl := pubSub.Elements().Child("subscriptions"); cmdEl != nil {
		if iq.IsGet() {
			opts := commandOptions{
				allowedAffiliations:  []string{pubsubmodel.Owner},
				includeSubscriptions: true,
				failOnNotFound:       true,
			}
			x.withCommandContext(func(cmdCtx *commandContext) {
				x.retrieveSubscriptions(cmdCtx, iq)
			}, opts, cmdEl, iq)
		} else if iq.IsSet() {
			opts := commandOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				failOnNotFound:      true,
			}
			x.withCommandContext(func(cmdCtx *commandContext) {
				x.updateSubscriptions(cmdCtx, cmdEl, iq)
			}, opts, cmdEl, iq)
		} else {
			_ = x.router.Route(iq.ServiceUnavailableError())
		}
		return
	}
	// Delete node
	if cmdEl := pubSub.Elements().Child("delete"); cmdEl != nil && iq.IsSet() {
		opts := commandOptions{
			allowedAffiliations:  []string{pubsubmodel.Owner},
			includeSubscriptions: true,
			failOnNotFound:       true,
		}
		x.withCommandContext(func(cmdCtx *commandContext) { x.delete(cmdCtx, iq) }, opts, cmdEl, iq)
		return
	}

	_ = x.router.Route(iq.FeatureNotImplementedError())
}

func (x *Pep) create(cmdCtx *commandContext, pubSubEl xmpp.XElement, iq *xmpp.IQ) {
	if cmdCtx.node != nil {
		_ = x.router.Route(iq.ConflictError())
		return
	}
	node := &pubsubmodel.Node{
		Host: cmdCtx.host,
		Name: cmdCtx.nodeID,
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
	if err := x.createNode(node); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	log.Infof("pep: created node (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) sendConfigurationForm(cmdCtx *commandContext, iq *xmpp.IQ) {
	// compose config form response
	configureNode := xmpp.NewElementName("configure")
	configureNode.SetAttribute("node", cmdCtx.nodeID)

	rosterGroups, err := storage.FetchRosterGroups(iq.ToJID().Node())
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}

	configureNode.AppendElement(cmdCtx.node.Options.Form(rosterGroups).Element())

	pubSubNode := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	pubSubNode.AppendElement(configureNode)

	res := iq.ResultIQ()
	res.AppendElement(pubSubNode)

	log.Infof("pep: sent configuration form (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(res)
}

func (x *Pep) configure(cmdCtx *commandContext, cmdElem xmpp.XElement, iq *xmpp.IQ) {
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
	cmdCtx.node.Options = *nodeOpts

	// update node config
	if err := storage.UpsertPubSubNode(cmdCtx.node); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// notify config update
	if cmdCtx.node.Options.DeliverNotifications && cmdCtx.node.Options.NotifyConfig {
		configElem := xmpp.NewElementName("configuration")
		configElem.SetAttribute("node", cmdCtx.nodeID)

		if cmdCtx.node.Options.DeliverPayloads {
			configElem.AppendElement(cmdCtx.node.Options.ResultForm().Element())
		}
		x.notifySubscribers(
			configElem,
			cmdCtx.subscriptions,
			cmdCtx.accessChecker,
			cmdCtx.host,
			cmdCtx.nodeID,
			cmdCtx.node.Options.NotificationType)
	}
	log.Infof("pep: node configuration updated (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) delete(cmdCtx *commandContext, iq *xmpp.IQ) {
	// delete node
	if err := storage.DeletePubSubNode(cmdCtx.host, cmdCtx.nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// notify delete
	if cmdCtx.node.Options.DeliverNotifications && cmdCtx.node.Options.NotifyDelete {
		deleteElem := xmpp.NewElementName("delete")
		deleteElem.SetAttribute("node", cmdCtx.nodeID)

		x.notifySubscribers(
			deleteElem,
			cmdCtx.subscriptions,
			cmdCtx.accessChecker,
			cmdCtx.host,
			cmdCtx.nodeID,
			cmdCtx.node.Options.NotificationType)
	}
	log.Infof("pep: deleted node (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) subscribe(cmdCtx *commandContext, cmdEl xmpp.XElement, iq *xmpp.IQ) {
	// validate JID portion
	subJID := cmdEl.Attributes().Get("jid")
	if subJID != iq.FromJID().ToBareJID().String() {
		_ = x.router.Route(invalidJIDError(iq))
		return
	}
	// create subscription
	subID := subscriptionID(subJID, cmdCtx.host, cmdCtx.nodeID)

	sub := pubsubmodel.Subscription{
		SubID:        subID,
		JID:          subJID,
		Subscription: pubsubmodel.Subscribed,
	}
	err := storage.UpsertPubSubNodeSubscription(&sub, cmdCtx.host, cmdCtx.nodeID)

	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	log.Infof("pep: subscription created (host: %s, node_id: %s, jid: %s)", cmdCtx.host, cmdCtx.nodeID, subJID)

	// notify subscription update
	subscriptionElem := xmpp.NewElementName("subscription")
	subscriptionElem.SetAttribute("node", cmdCtx.nodeID)
	subscriptionElem.SetAttribute("jid", subJID)
	subscriptionElem.SetAttribute("subid", subID)
	subscriptionElem.SetAttribute("subscription", pubsubmodel.Subscribed)

	if cmdCtx.node.Options.DeliverNotifications && cmdCtx.node.Options.NotifySub {
		x.notifyOwners(subscriptionElem, cmdCtx.affiliations, cmdCtx.host, cmdCtx.node.Options.NotificationType)
	}

	// send last node item
	switch cmdCtx.node.Options.SendLastPublishedItem {
	case pubsubmodel.OnSub, pubsubmodel.OnSubAndPresence:
		err := x.sendLastPublishedItem(sub, cmdCtx.accessChecker, cmdCtx.host, cmdCtx.nodeID, cmdCtx.node.Options.NotificationType)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
	}

	// compose response
	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	pubSubElem.AppendElement(subscriptionElem)
	iqRes.AppendElement(pubSubElem)

	_ = x.router.Route(iqRes)
}

func (x *Pep) unsubscribe(cmdCtx *commandContext, cmdEl xmpp.XElement, iq *xmpp.IQ) {
	subJID := cmdEl.Attributes().Get("jid")
	if subJID != iq.FromJID().ToBareJID().String() {
		_ = x.router.Route(iq.ForbiddenError())
		return
	}
	var subscription *pubsubmodel.Subscription
	for _, sub := range cmdCtx.subscriptions {
		if sub.JID == subJID {
			subscription = &sub
			break
		}
	}
	if subscription == nil {
		_ = x.router.Route(notSubscribedError(iq))
		return
	}
	// delete subscription
	if err := storage.DeletePubSubNodeSubscription(subJID, cmdCtx.host, cmdCtx.nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	log.Infof("pep: subscription removed (host: %s, node_id: %s, jid: %s)", cmdCtx.host, cmdCtx.nodeID, subJID)

	// notify subscription update
	subscriptionElem := xmpp.NewElementName("subscription")
	subscriptionElem.SetAttribute("node", cmdCtx.nodeID)
	subscriptionElem.SetAttribute("jid", subJID)
	subscriptionElem.SetAttribute("subid", subscription.SubID)
	subscriptionElem.SetAttribute("subscription", pubsubmodel.None)

	if cmdCtx.node.Options.DeliverNotifications && cmdCtx.node.Options.NotifySub {
		x.notifyOwners(subscriptionElem, cmdCtx.affiliations, cmdCtx.host, cmdCtx.node.Options.NotificationType)
	}

	// compose response
	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	pubSubElem.AppendElement(subscriptionElem)
	iqRes.AppendElement(pubSubElem)

	_ = x.router.Route(iqRes)
}

func (x *Pep) publish(cmdCtx *commandContext, cmdEl xmpp.XElement, iq *xmpp.IQ) {
	itemEl := cmdEl.Elements().Child("item")
	if itemEl == nil || len(itemEl.Elements().All()) != 1 {
		_ = x.router.Route(invalidPayloadError(iq))
		return
	}
	itemID := itemEl.Attributes().Get("id")
	if len(itemID) == 0 {
		// generate unique item identifier
		itemID = uuid.New().String()
	}
	// auto create node
	if cmdCtx.node == nil {
		if !cmdCtx.isAccountOwner {
			_ = x.router.Route(iq.ForbiddenError())
			return
		}
		cmdCtx.node = &pubsubmodel.Node{
			Host:    cmdCtx.host,
			Name:    cmdCtx.nodeID,
			Options: defaultNodeOptions,
		}
		cmdCtx.subscriptions = []pubsubmodel.Subscription{{
			JID:          cmdCtx.host,
			SubID:        subscriptionID(cmdCtx.host, cmdCtx.host, cmdCtx.nodeID),
			Subscription: pubsubmodel.Subscribed,
		}}
		if err := x.createNode(cmdCtx.node); err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
	}
	// persist node item
	if cmdCtx.node.Options.PersistItems {
		err := storage.UpsertPubSubNodeItem(&pubsubmodel.Item{
			ID:        itemID,
			Publisher: iq.FromJID().ToBareJID().String(),
			Payload:   itemEl.Elements().All()[0],
		}, cmdCtx.host, cmdCtx.nodeID, int(cmdCtx.node.Options.MaxItems))

		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
	}
	// notify published item
	notifyElem := xmpp.NewElementName("item")
	notifyElem.SetAttribute("id", itemID)

	if cmdCtx.node.Options.DeliverPayloads || !cmdCtx.node.Options.PersistItems {
		notifyElem.AppendElement(itemEl.Elements().All()[0])
	}
	x.notifySubscribers(
		notifyElem,
		cmdCtx.subscriptions,
		cmdCtx.accessChecker,
		cmdCtx.host,
		cmdCtx.nodeID,
		cmdCtx.node.Options.NotificationType)

	// compose response
	publishElem := xmpp.NewElementName("publish")
	publishElem.SetAttribute("node", cmdCtx.nodeID)
	resItemElem := xmpp.NewElementName("item")
	resItemElem.SetAttribute("id", itemID)
	publishElem.AppendElement(resItemElem)

	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	pubSubElem.AppendElement(publishElem)
	iqRes.AppendElement(pubSubElem)

	_ = x.router.Route(iqRes)
}

func (x *Pep) retrieveItems(cmdCtx *commandContext, cmdEl xmpp.XElement, iq *xmpp.IQ) {
	var itemIDs []string

	itemElems := cmdEl.Elements().Children("item")
	if len(itemElems) > 0 {
		for _, itemEl := range itemElems {
			itemID := itemEl.Attributes().Get("id")
			if len(itemID) == 0 {
				continue
			}
			itemIDs = append(itemIDs, itemID)
		}
	}
	// retrieve node items
	var items []pubsubmodel.Item
	var err error

	if len(itemIDs) > 0 {
		items, err = storage.FetchPubSubNodeItemsWithIDs(cmdCtx.host, cmdCtx.nodeID, itemIDs)
	} else {
		items, err = storage.FetchPubSubNodeItems(cmdCtx.host, cmdCtx.nodeID)
	}
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	log.Infof("pep: retrieved items (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	// compose response
	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	itemsElem := xmpp.NewElementName("items")
	itemsElem.SetAttribute("node", cmdCtx.nodeID)

	for _, itm := range items {
		itemElem := xmpp.NewElementName("item")
		itemElem.SetAttribute("id", itm.ID)
		itemElem.AppendElement(itm.Payload)

		itemsElem.AppendElement(itemsElem)
	}
	pubSubElem.AppendElement(itemsElem)
	iqRes.AppendElement(pubSubElem)

	_ = x.router.Route(iqRes)
}

func (x *Pep) retrieveAffiliations(cmdCtx *commandContext, iq *xmpp.IQ) {
	affiliationsElem := xmpp.NewElementName("affiliations")
	affiliationsElem.SetAttribute("node", cmdCtx.nodeID)

	for _, aff := range cmdCtx.affiliations {
		affElem := xmpp.NewElementName("affiliation")
		affElem.SetAttribute("jid", aff.JID)
		affElem.SetAttribute("affiliation", aff.Affiliation)
	}
	log.Infof("pep: retrieved affiliations (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	// compose response
	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	pubSubElem.AppendElement(affiliationsElem)
	iqRes.AppendElement(pubSubElem)

	_ = x.router.Route(iqRes)
}

func (x *Pep) updateAffiliations(cmdCtx *commandContext, cmdElem xmpp.XElement, iq *xmpp.IQ) {
	// update affiliations
	for _, affElem := range cmdElem.Elements().Children("affiliation") {
		var aff pubsubmodel.Affiliation
		aff.JID = affElem.Attributes().Get("jid")
		aff.Affiliation = affElem.Attributes().Get("affiliation")

		if aff.JID == cmdCtx.host {
			// ignore node owner affiliation update
			continue
		}
		var err error
		switch aff.Affiliation {
		case pubsubmodel.Owner, pubsubmodel.Member, pubsubmodel.Publisher, pubsubmodel.Outcast:
			err = storage.UpsertPubSubNodeAffiliation(&aff, cmdCtx.host, cmdCtx.nodeID)
		case pubsubmodel.None:
			err = storage.DeletePubSubNodeAffiliation(aff.JID, cmdCtx.host, cmdCtx.nodeID)
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
	log.Infof("pep: modified affiliations (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) retrieveSubscriptions(cmdCtx *commandContext, iq *xmpp.IQ) {
	subscriptionsElem := xmpp.NewElementName("subscriptions")
	subscriptionsElem.SetAttribute("node", cmdCtx.nodeID)

	for _, sub := range cmdCtx.subscriptions {
		subElem := xmpp.NewElementName("subscription")
		subElem.SetAttribute("subid", sub.SubID)
		subElem.SetAttribute("jid", sub.JID)
		subElem.SetAttribute("subscription", sub.Subscription)
	}
	log.Infof("pep: retrieved subscriptions (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	// compose response
	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	pubSubElem.AppendElement(subscriptionsElem)
	iqRes.AppendElement(pubSubElem)

	_ = x.router.Route(iqRes)
}

func (x *Pep) updateSubscriptions(cmdCtx *commandContext, cmdElem xmpp.XElement, iq *xmpp.IQ) {
	// update subscriptions
	for _, subElem := range cmdElem.Elements().Children("subscription") {
		var sub pubsubmodel.Subscription
		sub.SubID = subElem.Attributes().Get("subid")
		sub.JID = subElem.Attributes().Get("jid")
		sub.Subscription = subElem.Attributes().Get("subscription")

		if sub.JID == cmdCtx.host {
			// ignore node owner subscription update
			continue
		}
		var err error
		switch sub.Subscription {
		case pubsubmodel.Subscribed:
			err = storage.UpsertPubSubNodeSubscription(&sub, cmdCtx.host, cmdCtx.nodeID)
		case pubsubmodel.None:
			err = storage.DeletePubSubNodeSubscription(sub.JID, cmdCtx.host, cmdCtx.nodeID)
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
	log.Infof("pep: modified subscriptions (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) notifyOwners(notificationElem xmpp.XElement, affiliations []pubsubmodel.Affiliation, host, notificationType string) {
	hostJID, _ := jid.NewWithString(host, true)
	for _, affiliation := range affiliations {
		if affiliation.Affiliation != pubsubmodel.Owner {
			continue
		}
		toJID, _ := jid.NewWithString(affiliation.JID, true)
		eventMsg := eventMessage(notificationElem, hostJID, toJID, notificationType)

		_ = x.router.Route(eventMsg)
	}
}

func (x *Pep) notifySubscribers(
	notificationElem xmpp.XElement,
	subscriptions []pubsubmodel.Subscription,
	accessChecker *accessChecker,
	host string,
	nodeID string,
	notificationType string,
) {
	hostJID, _ := jid.NewWithString(host, true)
	for _, subscription := range subscriptions {
		if subscription.Subscription != pubsubmodel.Subscribed {
			continue
		}
		if subscription.JID != host {
			// check JID access before notifying
			err := accessChecker.checkAccess(host, subscription.JID)
			switch err {
			case nil:
				break
			case errPresenceSubscriptionRequired, errNotInRosterGroup, errNotOnWhiteList:
				continue
			default:
				log.Error(err)
				continue
			}
		}
		subscriberJID, _ := jid.NewWithString(subscription.JID, true)

		if pp := x.onlinePresenceProvider; pp != nil {
			onlinePresences := pp.OnlinePresencesMatchingJID(subscriberJID)

			for _, onlinePresence := range onlinePresences {
				presence := onlinePresence.Presence
				if caps := onlinePresence.Caps; caps != nil {
					if !caps.HasFeature(nodeID + "+notify") {
						continue
					}
				}
				// notify to full jid
				eventMsg := eventMessage(notificationElem, hostJID, presence.FromJID(), notificationType)
				_ = x.router.Route(eventMsg)
			}
		} else {
			// broadcast event message
			eventMsg := eventMessage(notificationElem, hostJID, subscriberJID, notificationType)
			_ = x.router.Route(eventMsg)
		}
	}
}

func (x *Pep) withCommandContext(fn func(cmdCtx *commandContext), opts commandOptions, cmdElem xmpp.XElement, iq *xmpp.IQ) {
	var ctx commandContext

	nodeID := cmdElem.Attributes().Get("node")
	if len(nodeID) == 0 {
		_ = x.router.Route(nodeIDRequiredError(iq))
		return
	}
	fromJID := iq.FromJID().ToBareJID().String()
	host := iq.ToJID().ToBareJID().String()

	ctx.host = host
	ctx.nodeID = nodeID
	ctx.isAccountOwner = fromJID == host

	// fetch node
	node, err := storage.FetchPubSubNode(host, nodeID)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	if node == nil {
		if opts.failOnNotFound {
			_ = x.router.Route(iq.ItemNotFoundError())
		} else {
			fn(&ctx)
		}
		return
	}
	ctx.node = node

	// fetch affiliations
	var affiliations []pubsubmodel.Affiliation

	if len(opts.allowedAffiliations) > 0 || opts.includeAffiliations || opts.checkAccess {
		affiliations, err = storage.FetchPubSubNodeAffiliations(host, nodeID)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
	}
	ctx.accessChecker = &accessChecker{
		accessModel:         node.Options.AccessModel,
		rosterAllowedGroups: node.Options.RosterGroupsAllowed,
		affiliations:        affiliations,
	}
	// check access
	if opts.checkAccess && !ctx.isAccountOwner {
		for _, aff := range affiliations {
			if aff.JID == fromJID && aff.Affiliation == pubsubmodel.Outcast {
				_ = x.router.Route(iq.ForbiddenError())
				return
			}
		}
		err := ctx.accessChecker.checkAccess(host, fromJID)
		switch err {
		case errPresenceSubscriptionRequired:
			_ = x.router.Route(presenceSubscriptionRequiredError(iq))
			return

		case errNotInRosterGroup:
			_ = x.router.Route(notInRosterGroupError(iq))
			return

		case errNotOnWhiteList:
			_ = x.router.Route(notOnWhitelistError(iq))
			return

		default:
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
	}
	// validate affiliation
	if len(opts.allowedAffiliations) > 0 {
		fromJID := iq.FromJID().ToBareJID().String()

		var allowed bool
		for _, aff := range affiliations {
			if aff.JID != fromJID {
				continue
			}
			for _, allowedAff := range opts.allowedAffiliations {
				if allowedAff != aff.Affiliation {
					continue
				}
				allowed = true
				break
			}
			break
		}
		if !allowed {
			_ = x.router.Route(iq.ForbiddenError())
			return
		}
	}
	if opts.includeAffiliations {
		ctx.affiliations = affiliations
	}

	// fetch subscriptions
	if opts.includeSubscriptions {
		subscriptions, err := storage.FetchPubSubNodeSubscriptions(host, nodeID)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
		ctx.subscriptions = subscriptions
	}
	fn(&ctx)
}

func (x *Pep) createNode(node *pubsubmodel.Node) error {
	// create node
	if err := storage.UpsertPubSubNode(node); err != nil {
		return err
	}
	// create owner affiliation
	ownerAffiliation := &pubsubmodel.Affiliation{
		JID:         node.Host,
		Affiliation: pubsubmodel.Owner,
	}
	if err := storage.UpsertPubSubNodeAffiliation(ownerAffiliation, node.Host, node.Name); err != nil {
		return err
	}
	// create owner subscription
	ownerSub := &pubsubmodel.Subscription{
		SubID:        subscriptionID(node.Host, node.Host, node.Name),
		JID:          node.Host,
		Subscription: pubsubmodel.Subscribed,
	}
	return storage.UpsertPubSubNodeSubscription(ownerSub, node.Host, node.Name)
}

func (x *Pep) sendLastPublishedItem(sub pubsubmodel.Subscription, accessChecker *accessChecker, host, nodeID, notificationType string) error {
	items, err := storage.FetchPubSubNodeItems(host, nodeID)
	if err != nil {
		return err
	}
	if len(items) > 0 {
		lastItem := items[len(items)-1]

		itemsEl := xmpp.NewElementName("items")
		itemsEl.SetAttribute("node", nodeID)
		itemsEl.AppendElement(lastItem.Payload)

		x.notifySubscribers(
			itemsEl,
			[]pubsubmodel.Subscription{sub},
			accessChecker,
			host,
			nodeID,
			notificationType)
	}
	return nil
}

func eventMessage(payloadElem xmpp.XElement, hostJID, toJID *jid.JID, notificationType string) *xmpp.Message {
	msg := xmpp.NewMessageType(uuid.New().String(), notificationType)
	msg.SetFromJID(hostJID)
	msg.SetToJID(toJID)
	eventElem := xmpp.NewElementNamespace("event", pubSubEventNamespace)
	eventElem.AppendElement(payloadElem)
	msg.AppendElement(eventElem)

	return msg
}

func nodeIDRequiredError(stanza xmpp.Stanza) xmpp.Stanza {
	errorElements := []xmpp.XElement{xmpp.NewElementNamespace("nodeid-required", pubSubErrorNamespace)}
	return xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrNotAcceptable, errorElements)
}

func invalidPayloadError(stanza xmpp.Stanza) xmpp.Stanza {
	errorElements := []xmpp.XElement{xmpp.NewElementNamespace("invalid-payload", pubSubErrorNamespace)}
	return xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrBadRequest, errorElements)
}

func invalidJIDError(stanza xmpp.Stanza) xmpp.Stanza {
	errorElements := []xmpp.XElement{xmpp.NewElementNamespace("invalid-jid", pubSubErrorNamespace)}
	return xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrBadRequest, errorElements)
}

func presenceSubscriptionRequiredError(stanza xmpp.Stanza) xmpp.Stanza {
	errorElements := []xmpp.XElement{xmpp.NewElementNamespace("presence-subscription-required", pubSubErrorNamespace)}
	return xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrNotAuthorized, errorElements)
}

func notInRosterGroupError(stanza xmpp.Stanza) xmpp.Stanza {
	errorElements := []xmpp.XElement{xmpp.NewElementNamespace("not-in-roster-group", pubSubErrorNamespace)}
	return xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrNotAuthorized, errorElements)
}

func notOnWhitelistError(stanza xmpp.Stanza) xmpp.Stanza {
	errorElements := []xmpp.XElement{xmpp.NewElementNamespace("closed-node", pubSubErrorNamespace)}
	return xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrNotAllowed, errorElements)
}

func notSubscribedError(stanza xmpp.Stanza) xmpp.Stanza {
	errorElements := []xmpp.XElement{xmpp.NewElementNamespace("not-subscribed", pubSubErrorNamespace)}
	return xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrUnexpectedRequest, errorElements)
}

func subscriptionID(jid, host, name string) string {
	h := sha256.New()
	h.Write([]byte(jid + host + name))
	return fmt.Sprintf("%x", h.Sum(nil))
}
