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
	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/module/roster/presencehub"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/runqueue"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

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
	PersistItems:          true,
	AccessModel:           pubsubmodel.Presence,
	PublishModel:          pubsubmodel.Publishers,
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

type Pep struct {
	router      *router.Router
	runQueue    *runqueue.RunQueue
	presenceHub *presencehub.PresenceHub
}

func New(disco *xep0030.DiscoInfo, presenceHub *presencehub.PresenceHub, router *router.Router) *Pep {
	p := &Pep{
		router:      router,
		runQueue:    runqueue.New("xep0163"),
		presenceHub: presenceHub,
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

// ProcessIQ processes a version IQ taking according actions over the associated stream
func (x *Pep) ProcessIQ(iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		x.processIQ(iq)
	})
}

// SubscribeToAll subscribes a jid to all host nodes
func (x *Pep) SubscribeToAll(host string, jid *jid.JID) {
	x.runQueue.Run(func() {
		if err := x.subscribeToAll(host, jid); err != nil {
			log.Error(err)
		}
	})
}

// UnsubscribeFromAll unsubscribes a jid from all host nodes
func (x *Pep) UnsubscribeFromAll(host string, jid *jid.JID) {
	x.runQueue.Run(func() {
		if err := x.unsubscribeFromAll(host, jid); err != nil {
			log.Error(err)
		}
	})
}

// DeliverLastItems delivers last items from all those nodes to which the jid is subscribed
func (x *Pep) DeliverLastItems(jid *jid.JID) {
	x.runQueue.Run(func() {
		if err := x.deliverLastItems(jid); err != nil {
			log.Error(err)
		}
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

func (x *Pep) subscribeToAll(host string, subJID *jid.JID) error {
	nodes, err := storage.FetchNodes(host)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		// upsert subscription
		subID := subscriptionID(subJID.ToBareJID().String(), host, n.Name)
		sub := pubsubmodel.Subscription{
			SubID:        subID,
			JID:          subJID.ToBareJID().String(),
			Subscription: pubsubmodel.Subscribed,
		}
		if err := storage.UpsertNodeSubscription(&sub, host, n.Name); err != nil {
			return err
		}
		log.Infof("pep: subscription created (host: %s, node_id: %s, jid: %s)", host, n.Name, subJID)

		// notify subscription update
		affiliations, err := storage.FetchNodeAffiliations(host, n.Name)
		if err != nil {
			return err
		}
		subscriptionElem := xmpp.NewElementName("subscription")
		subscriptionElem.SetAttribute("node", n.Name)
		subscriptionElem.SetAttribute("jid", subJID.ToBareJID().String())
		subscriptionElem.SetAttribute("subid", subID)
		subscriptionElem.SetAttribute("subscription", pubsubmodel.Subscribed)

		if n.Options.DeliverNotifications && n.Options.NotifySub {
			x.notifyOwners(subscriptionElem, affiliations, host, n.Options.NotificationType)
		}
		// send last node item
		switch n.Options.SendLastPublishedItem {
		case pubsubmodel.OnSub, pubsubmodel.OnSubAndPresence:
			var subAff *pubsubmodel.Affiliation
			for _, aff := range affiliations {
				if aff.JID == subJID.ToBareJID().String() {
					subAff = &aff
					break
				}
			}
			accessChecker := &accessChecker{
				host:                n.Host,
				nodeID:              n.Name,
				accessModel:         n.Options.AccessModel,
				rosterAllowedGroups: n.Options.RosterGroupsAllowed,
				affiliation:         subAff,
			}
			if err := x.sendLastPublishedItem(subJID, accessChecker, host, n.Name, n.Options.NotificationType); err != nil {
				return err
			}
		}
	}
	return nil
}

func (x *Pep) unsubscribeFromAll(host string, subJID *jid.JID) error {
	nodes, err := storage.FetchNodes(host)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		if err := storage.DeleteNodeSubscription(subJID.ToBareJID().String(), host, n.Name); err != nil {
			return err
		}
		log.Infof("pep: subscription removed (host: %s, node_id: %s, jid: %s)", host, n.Name, subJID.ToBareJID().String())
	}
	return nil
}

func (x *Pep) deliverLastItems(jid *jid.JID) error {
	nodes, err := storage.FetchSubscribedNodes(jid.ToBareJID().String())
	if err != nil {
		return err
	}
	for _, node := range nodes {
		if node.Options.SendLastPublishedItem != pubsubmodel.OnSubAndPresence {
			continue
		}
		aff, err := storage.FetchNodeAffiliation(node.Host, node.Name, jid.ToBareJID().String())
		if err != nil {
			return err
		}
		accessChecker := &accessChecker{
			host:                node.Host,
			nodeID:              node.Name,
			accessModel:         node.Options.AccessModel,
			rosterAllowedGroups: node.Options.RosterGroupsAllowed,
			affiliation:         aff,
		}
		if err := x.sendLastPublishedItem(jid, accessChecker, node.Host, node.Name, node.Options.NotificationType); err != nil {
			return err
		}
		log.Infof("pep: delivered last items: %s (node: %s, host: %s)", jid.String(), node.Host, node.Name)
	}
	return nil
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
			x.withCommandContext(func(cmdCtx *commandContext) { x.sendConfigurationForm(cmdCtx, iq) }, opts, cmdEl, iq)
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
	if err := storage.UpsertNode(cmdCtx.node); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// notify config update
	opts := cmdCtx.node.Options

	if opts.DeliverNotifications && opts.NotifyConfig {
		configElem := xmpp.NewElementName("configuration")
		configElem.SetAttribute("node", cmdCtx.nodeID)

		if opts.DeliverPayloads {
			configElem.AppendElement(opts.ResultForm().Element())
		}
		x.notifySubscribers(
			configElem,
			cmdCtx.subscriptions,
			cmdCtx.accessChecker,
			cmdCtx.host,
			cmdCtx.nodeID,
			opts.NotificationType)
	}
	log.Infof("pep: node configuration updated (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) delete(cmdCtx *commandContext, iq *xmpp.IQ) {
	// delete node
	if err := storage.DeleteNode(cmdCtx.host, cmdCtx.nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// notify delete
	opts := cmdCtx.node.Options

	if opts.DeliverNotifications && opts.NotifyDelete {
		deleteElem := xmpp.NewElementName("delete")
		deleteElem.SetAttribute("node", cmdCtx.nodeID)

		x.notifySubscribers(
			deleteElem,
			cmdCtx.subscriptions,
			cmdCtx.accessChecker,
			cmdCtx.host,
			cmdCtx.nodeID,
			opts.NotificationType)
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
	err := storage.UpsertNodeSubscription(&sub, cmdCtx.host, cmdCtx.nodeID)

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

	opts := cmdCtx.node.Options
	if opts.DeliverNotifications && opts.NotifySub {
		x.notifyOwners(subscriptionElem, cmdCtx.affiliations, cmdCtx.host, opts.NotificationType)
	}
	// send last node item
	switch opts.SendLastPublishedItem {
	case pubsubmodel.OnSub, pubsubmodel.OnSubAndPresence:
		subscriberJID, _ := jid.NewWithString(sub.JID, true)
		err := x.sendLastPublishedItem(subscriberJID, cmdCtx.accessChecker, cmdCtx.host, cmdCtx.nodeID, opts.NotificationType)
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
	if err := storage.DeleteNodeSubscription(subJID, cmdCtx.host, cmdCtx.nodeID); err != nil {
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

	opts := cmdCtx.node.Options
	if opts.DeliverNotifications && opts.NotifySub {
		x.notifyOwners(subscriptionElem, cmdCtx.affiliations, cmdCtx.host, opts.NotificationType)
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
	opts := cmdCtx.node.Options
	if opts.PersistItems {
		err := storage.UpsertNodeItem(&pubsubmodel.Item{
			ID:        itemID,
			Publisher: iq.FromJID().ToBareJID().String(),
			Payload:   itemEl.Elements().All()[0],
		}, cmdCtx.host, cmdCtx.nodeID, int(opts.MaxItems))

		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
	}
	log.Infof("pep: published item (host: %s, node_id: %s, item_id: %s)", cmdCtx.host, cmdCtx.nodeID, itemID)

	// notify published item
	notifyElem := xmpp.NewElementName("item")
	notifyElem.SetAttribute("id", itemID)

	if opts.DeliverPayloads || !opts.PersistItems {
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
		items, err = storage.FetchNodeItemsWithIDs(cmdCtx.host, cmdCtx.nodeID, itemIDs)
	} else {
		items, err = storage.FetchNodeItems(cmdCtx.host, cmdCtx.nodeID)
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

		itemsElem.AppendElement(itemElem)
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

		affiliationsElem.AppendElement(affElem)
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
			err = storage.UpsertNodeAffiliation(&aff, cmdCtx.host, cmdCtx.nodeID)
		case pubsubmodel.None:
			err = storage.DeleteNodeAffiliation(aff.JID, cmdCtx.host, cmdCtx.nodeID)
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

		subscriptionsElem.AppendElement(subElem)
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
			err = storage.UpsertNodeSubscription(&sub, cmdCtx.host, cmdCtx.nodeID)
		case pubsubmodel.None:
			err = storage.DeleteNodeSubscription(sub.JID, cmdCtx.host, cmdCtx.nodeID)
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
	subscribers []pubsubmodel.Subscription,
	accessChecker *accessChecker,
	host string,
	nodeID string,
	notificationType string,
) {
	var toJIDs []jid.JID
	for _, subscriber := range subscribers {
		if subscriber.Subscription != pubsubmodel.Subscribed {
			continue
		}
		subscriberJID, _ := jid.NewWithString(subscriber.JID, true)
		toJIDs = append(toJIDs, *subscriberJID)
	}
	x.notify(notificationElem, toJIDs, accessChecker, host, nodeID, notificationType)
}

func (x *Pep) notify(
	notificationElem xmpp.XElement,
	toJIDs []jid.JID,
	accessChecker *accessChecker,
	host string,
	nodeID string,
	notificationType string,
) {
	hostJID, _ := jid.NewWithString(host, true)
	for _, toJID := range toJIDs {
		if toJID.ToBareJID().String() != host {
			// check JID access before notifying
			err := accessChecker.checkAccess(host, toJID.ToBareJID().String())
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

		if ph := x.presenceHub; ph != nil {
			onlinePresences := ph.AvailablePresencesMatchingJID(&toJID)

			for _, onlinePresence := range onlinePresences {
				caps := onlinePresence.Caps
				if caps == nil {
					goto broadcastEventMsg // broadcast event message
				}
				if !caps.HasFeature(nodeID + "+notify") {
					continue
				}
				// notify to full jid
				presence := onlinePresence.Presence

				eventMsg := eventMessage(notificationElem, hostJID, presence.FromJID(), notificationType)
				_ = x.router.Route(eventMsg)
			}
			return
		}
	broadcastEventMsg:
		// broadcast event message
		eventMsg := eventMessage(notificationElem, hostJID, &toJID, notificationType)
		_ = x.router.Route(eventMsg)
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
	node, err := storage.FetchNode(host, nodeID)
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

	// fetch affiliation
	aff, err := storage.FetchNodeAffiliation(host, nodeID, fromJID)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	ctx.accessChecker = &accessChecker{
		host:                node.Host,
		nodeID:              node.Name,
		accessModel:         node.Options.AccessModel,
		rosterAllowedGroups: node.Options.RosterGroupsAllowed,
		affiliation:         aff,
	}
	// check access
	if opts.checkAccess && !ctx.isAccountOwner {
		err := ctx.accessChecker.checkAccess(host, fromJID)
		switch err {
		case nil:
			break

		case errOutcastMember:
			_ = x.router.Route(iq.ForbiddenError())
			return

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
		var allowed bool
		for _, allowedAff := range opts.allowedAffiliations {
			if aff != nil && aff.Affiliation == allowedAff {
				allowed = true
				break
			}
		}
		if !allowed {
			_ = x.router.Route(iq.ForbiddenError())
			return
		}
	}
	// fetch subscriptions
	if opts.includeSubscriptions {
		subscriptions, err := storage.FetchNodeSubscriptions(host, nodeID)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
		ctx.subscriptions = subscriptions
	}
	// fetch affiliations
	if opts.includeAffiliations {
		affiliations, err := storage.FetchNodeAffiliations(host, nodeID)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
		ctx.affiliations = affiliations
	}
	fn(&ctx)
}

func (x *Pep) createNode(node *pubsubmodel.Node) error {
	// create node
	if err := storage.UpsertNode(node); err != nil {
		return err
	}
	// create owner affiliation
	ownerAffiliation := &pubsubmodel.Affiliation{
		JID:         node.Host,
		Affiliation: pubsubmodel.Owner,
	}
	if err := storage.UpsertNodeAffiliation(ownerAffiliation, node.Host, node.Name); err != nil {
		return err
	}
	// create owner subscription
	ownerSub := &pubsubmodel.Subscription{
		SubID:        subscriptionID(node.Host, node.Host, node.Name),
		JID:          node.Host,
		Subscription: pubsubmodel.Subscribed,
	}
	return storage.UpsertNodeSubscription(ownerSub, node.Host, node.Name)
}

func (x *Pep) sendLastPublishedItem(toJID *jid.JID, accessChecker *accessChecker, host, nodeID, notificationType string) error {
	lastItem, err := storage.FetchNodeLastItem(host, nodeID)
	if err != nil {
		return err
	}
	if lastItem == nil {
		return nil
	}
	itemsEl := xmpp.NewElementName("items")
	itemsEl.SetAttribute("node", nodeID)
	itemsEl.AppendElement(lastItem.Payload)

	x.notify(
		itemsEl,
		[]jid.JID{*toJID},
		accessChecker,
		host,
		nodeID,
		notificationType)
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
