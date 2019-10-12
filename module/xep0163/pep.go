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
	rostermodel "github.com/ortuman/jackal/model/roster"
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

type nodeContextFetchOptions struct {
	allowedAffiliations  []string
	includeAffiliations  bool
	includeSubscriptions bool
	failOnNotFound       bool
}

type nodeContext struct {
	host          string
	nodeID        string
	node          *pubsubmodel.Node
	affiliations  []pubsubmodel.Affiliation
	subscriptions []pubsubmodel.Subscription
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

func (x *Pep) processRequest(iq *xmpp.IQ, pubSubEl xmpp.XElement) {
	// Create node
	// https://xmpp.org/extensions/xep-0060.html#owner-create
	if cmdEl := pubSubEl.Elements().Child("create"); cmdEl != nil && iq.IsSet() {
		x.withNodeContext(func(ni *nodeContext) { x.createNode(ni, pubSubEl, iq) }, nodeContextFetchOptions{}, cmdEl, iq)
		return
	}

	// Subscribe
	// https://xmpp.org/extensions/xep-0060.html#subscriber-subscribe
	if cmdEl := pubSubEl.Elements().Child("subscribe"); cmdEl != nil && iq.IsSet() {
		opts := nodeContextFetchOptions{
			failOnNotFound: true,
		}
		x.withNodeContext(func(ni *nodeContext) { x.subscribe(ni, cmdEl, iq) }, opts, cmdEl, iq)
		return
	}

	_ = x.router.Route(iq.FeatureNotImplementedError())
}

func (x *Pep) processOwnerRequest(iq *xmpp.IQ, pubSub xmpp.XElement) {
	// Configure node
	// https://xmpp.org/extensions/xep-0060.html#owner-configure
	if cmdEl := pubSub.Elements().Child("configure"); cmdEl != nil {
		if iq.IsGet() {
			// send configuration form
			opts := nodeContextFetchOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				failOnNotFound:      true,
			}
			x.withNodeContext(func(ni *nodeContext) { x.sendConfigurationForm(ni, iq) }, opts, cmdEl, iq)
		} else if iq.IsSet() {
			// update node configuration
			opts := nodeContextFetchOptions{
				allowedAffiliations:  []string{pubsubmodel.Owner},
				includeSubscriptions: true,
				failOnNotFound:       true,
			}
			x.withNodeContext(func(nCtx *nodeContext) { x.configureNode(nCtx, cmdEl, iq) }, opts, cmdEl, iq)
		} else {
			_ = x.router.Route(iq.ServiceUnavailableError())
		}
		return
	}

	// Manage affiliations
	// https://xmpp.org/extensions/xep-0060.html#owner-affiliations
	if cmdEl := pubSub.Elements().Child("affiliations"); cmdEl != nil {
		if iq.IsGet() {
			opts := nodeContextFetchOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				includeAffiliations: true,
				failOnNotFound:      true,
			}
			x.withNodeContext(func(nCtx *nodeContext) { x.retrieveAffiliations(nCtx, iq) }, opts, cmdEl, iq)
		} else if iq.IsSet() {
			opts := nodeContextFetchOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				failOnNotFound:      true,
			}
			x.withNodeContext(func(nCtx *nodeContext) { x.updateAffiliations(nCtx, cmdEl, iq) }, opts, cmdEl, iq)
		} else {
			_ = x.router.Route(iq.ServiceUnavailableError())
		}
		return
	}
	// Manage subscriptions
	// https://xmpp.org/extensions/xep-0060.html#owner-subscriptions
	if cmdEl := pubSub.Elements().Child("subscriptions"); cmdEl != nil {
		if iq.IsGet() {
			opts := nodeContextFetchOptions{
				allowedAffiliations:  []string{pubsubmodel.Owner},
				includeSubscriptions: true,
				failOnNotFound:       true,
			}
			x.withNodeContext(func(nCtx *nodeContext) { x.retrieveSubscriptions(nCtx, iq) }, opts, cmdEl, iq)
		} else if iq.IsSet() {
			opts := nodeContextFetchOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				failOnNotFound:      true,
			}
			x.withNodeContext(func(nCtx *nodeContext) { x.updateSubscriptions(nCtx, cmdEl, iq) }, opts, cmdEl, iq)
		} else {
			_ = x.router.Route(iq.ServiceUnavailableError())
		}
		return
	}

	// Delete node
	// https://xmpp.org/extensions/xep-0060.html#owner-delete
	if cmdEl := pubSub.Elements().Child("delete"); cmdEl != nil && iq.IsSet() {
		opts := nodeContextFetchOptions{
			allowedAffiliations:  []string{pubsubmodel.Owner},
			includeSubscriptions: true,
			failOnNotFound:       true,
		}
		x.withNodeContext(func(nCtx *nodeContext) { x.deleteNode(nCtx, iq) }, opts, cmdEl, iq)
		return
	}

	_ = x.router.Route(iq.FeatureNotImplementedError())
}

func (x *Pep) createNode(nCtx *nodeContext, pubSubEl xmpp.XElement, iq *xmpp.IQ) {
	if nCtx.node != nil {
		_ = x.router.Route(iq.ConflictError())
		return
	}
	node := &pubsubmodel.Node{
		Host: nCtx.host,
		Name: nCtx.nodeID,
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
	log.Infof("pep: created node (host: %s, node_id: %s)", nCtx.host, nCtx.nodeID)

	// create owner affiliation
	ownerAffiliation := &pubsubmodel.Affiliation{
		JID:         nCtx.host,
		Affiliation: pubsubmodel.Owner,
	}
	if err := storage.UpsertPubSubNodeAffiliation(ownerAffiliation, nCtx.host, nCtx.nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// create owner subscription
	ownerSub := &pubsubmodel.Subscription{
		SubID:        subscriptionID(nCtx.host, pubsubmodel.Subscribed, nCtx.host, nCtx.nodeID),
		JID:          nCtx.host,
		Subscription: pubsubmodel.Subscribed,
	}
	if err := storage.UpsertPubSubNodeSubscription(ownerSub, nCtx.host, nCtx.nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}

	// reply
	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) sendConfigurationForm(nCtx *nodeContext, iq *xmpp.IQ) {
	// compose config form response
	configureNode := xmpp.NewElementName("configure")
	configureNode.SetAttribute("node", nCtx.nodeID)

	rosterGroups, err := storage.FetchRosterGroups(iq.ToJID().Node())
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}

	configureNode.AppendElement(nCtx.node.Options.Form(rosterGroups).Element())

	pubSubNode := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	pubSubNode.AppendElement(configureNode)

	res := iq.ResultIQ()
	res.AppendElement(pubSubNode)

	log.Infof("pep: sent configuration form (host: %s, node_id: %s)", nCtx.host, nCtx.nodeID)

	// reply
	_ = x.router.Route(res)
}

func (x *Pep) configureNode(nCtx *nodeContext, cmdElem xmpp.XElement, iq *xmpp.IQ) {
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
	nCtx.node.Options = *nodeOpts

	// update node config
	if err := storage.UpsertPubSubNode(nCtx.node); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// notify config update
	if nCtx.node.Options.DeliverNotifications && nCtx.node.Options.NotifyConfig {
		configElem := xmpp.NewElementName("configuration")
		configElem.SetAttribute("node", nCtx.nodeID)

		if nCtx.node.Options.DeliverPayloads {
			configElem.AppendElement(nCtx.node.Options.ResultForm().Element())
		}
		x.notify(configElem, nCtx.subscriptions, nCtx.host)
	}
	log.Infof("pep: node configuration updated (host: %s, node_id: %s)", nCtx.host, nCtx.nodeID)

	// reply
	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) deleteNode(nCtx *nodeContext, iq *xmpp.IQ) {
	// delete node
	if err := storage.DeletePubSubNode(nCtx.host, nCtx.nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	// notify delete
	if nCtx.node.Options.DeliverNotifications && nCtx.node.Options.NotifyDelete {
		deleteElem := xmpp.NewElementName("delete")
		deleteElem.SetAttribute("node", nCtx.nodeID)

		x.notify(deleteElem, nCtx.subscriptions, nCtx.host)
	}
	log.Infof("pep: deleted node (host: %s, node_id: %s)", nCtx.host, nCtx.nodeID)

	// reply
	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) subscribe(nCtx *nodeContext, cmdEl xmpp.XElement, iq *xmpp.IQ) {
	// validate JID portion
	subJID := cmdEl.Attributes().Get("jid")
	if subJID != iq.FromJID().ToBareJID().String() {
		_ = x.router.Route(invalidJIDError(iq))
		return
	}

	// check access
	for _, aff := range nCtx.affiliations {
		if aff.JID == subJID && aff.Affiliation == pubsubmodel.Outcast {
			_ = x.router.Route(iq.ForbiddenError())
			return
		}
	}

	switch nCtx.node.Options.AccessModel {
	case pubsubmodel.Open:
		break

	case pubsubmodel.Presence:
		allowed, err := checkPresenceAccess(nCtx.host, subJID)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
		if !allowed {
			_ = x.router.Route(presenceSubscriptionRequiredError(iq))
			return
		}

	case pubsubmodel.Roster:
		allowed, err := checkRosterAccess(nCtx.host, subJID, nCtx.node.Options.RosterGroupsAllowed)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(iq.InternalServerError())
			return
		}
		if !allowed {
			_ = x.router.Route(notInRosterGroupError(iq))
			return
		}

	case pubsubmodel.WhiteList:
		if !checkWhitelistAccess(nCtx.host, subJID, nCtx.affiliations) {
			_ = x.router.Route(notOnWhitelistError(iq))
			return
		}
	}

	// create subscription
	subID := subscriptionID(subJID, pubsubmodel.Subscribed, nCtx.host, nCtx.nodeID)

	err := storage.UpsertPubSubNodeSubscription(&pubsubmodel.Subscription{
		SubID:        subID,
		JID:          subJID,
		Subscription: pubsubmodel.Subscribed,
	}, nCtx.host, nCtx.nodeID)

	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}

	// reply
	subscriptionElem := xmpp.NewElementName("subscription")
	subscriptionElem.SetAttribute("node", nCtx.nodeID)
	subscriptionElem.SetAttribute("jid", subJID)
	subscriptionElem.SetAttribute("subid", subID)
	subscriptionElem.SetAttribute("subscription", pubsubmodel.Subscribed)

	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	pubSubElem.AppendElement(subscriptionElem)
	iqRes.AppendElement(pubSubElem)
}

func (x *Pep) retrieveAffiliations(nCtx *nodeContext, iq *xmpp.IQ) {
	// compose response
	affiliationsElem := xmpp.NewElementName("affiliations")
	affiliationsElem.SetAttribute("node", nCtx.nodeID)

	for _, aff := range nCtx.affiliations {
		affElem := xmpp.NewElementName("affiliation")
		affElem.SetAttribute("jid", aff.JID)
		affElem.SetAttribute("affiliation", aff.Affiliation)
	}
	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	pubSubElem.AppendElement(affiliationsElem)
	iqRes.AppendElement(pubSubElem)

	log.Infof("pep: retrieved affiliations (host: %s, node_id: %s)", nCtx.host, nCtx.nodeID)

	// reply
	_ = x.router.Route(iqRes)
}

func (x *Pep) updateAffiliations(nCtx *nodeContext, cmdElem xmpp.XElement, iq *xmpp.IQ) {
	// update affiliations
	for _, affElem := range cmdElem.Elements().Children("affiliation") {
		var aff pubsubmodel.Affiliation
		aff.JID = affElem.Attributes().Get("jid")
		aff.Affiliation = affElem.Attributes().Get("affiliation")

		if aff.JID == nCtx.host {
			// ignore node owner affiliation update
			continue
		}
		var err error
		switch aff.Affiliation {
		case pubsubmodel.Owner, pubsubmodel.Member, pubsubmodel.Publisher, pubsubmodel.Outcast:
			err = storage.UpsertPubSubNodeAffiliation(&aff, nCtx.host, nCtx.nodeID)
		case pubsubmodel.None:
			err = storage.DeletePubSubNodeAffiliation(aff.JID, nCtx.host, nCtx.nodeID)
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
	log.Infof("pep: modified affiliations (host: %s, node_id: %s)", nCtx.host, nCtx.nodeID)

	// reply
	_ = x.router.Route(iq.ResultIQ())
}

func (x *Pep) retrieveSubscriptions(nCtx *nodeContext, iq *xmpp.IQ) {
	// compose response
	subscriptionsElem := xmpp.NewElementName("subscriptions")
	subscriptionsElem.SetAttribute("node", nCtx.nodeID)

	for _, sub := range nCtx.subscriptions {
		subElem := xmpp.NewElementName("subscription")
		subElem.SetAttribute("subid", sub.SubID)
		subElem.SetAttribute("jid", sub.JID)
		subElem.SetAttribute("subscription", sub.Subscription)
	}
	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	pubSubElem.AppendElement(subscriptionsElem)
	iqRes.AppendElement(pubSubElem)

	log.Infof("pep: retrieved subscriptions (host: %s, node_id: %s)", nCtx.host, nCtx.nodeID)

	// reply
	_ = x.router.Route(iqRes)
}

func (x *Pep) updateSubscriptions(nCtx *nodeContext, cmdElem xmpp.XElement, iq *xmpp.IQ) {
	// update subscriptions
	for _, subElem := range cmdElem.Elements().Children("subscription") {
		var sub pubsubmodel.Subscription
		sub.SubID = subElem.Attributes().Get("subid")
		sub.JID = subElem.Attributes().Get("jid")
		sub.Subscription = subElem.Attributes().Get("subscription")

		if sub.JID == nCtx.host {
			// ignore node owner subscription update
			continue
		}
		var err error
		switch sub.Subscription {
		case pubsubmodel.Subscribed:
			err = storage.UpsertPubSubNodeSubscription(&sub, nCtx.host, nCtx.nodeID)
		case pubsubmodel.None:
			err = storage.DeletePubSubNodeSubscription(sub.JID, nCtx.host, nCtx.nodeID)
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
	log.Infof("pep: modified subscriptions (host: %s, node_id: %s)", nCtx.host, nCtx.nodeID)

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

func (x *Pep) withNodeContext(fn func(nCtx *nodeContext), opts nodeContextFetchOptions, cmdElem xmpp.XElement, iq *xmpp.IQ) {
	var ctx nodeContext

	nodeID := cmdElem.Attributes().Get("node")
	if len(nodeID) == 0 {
		_ = x.router.Route(nodeIDRequiredError(iq))
		return
	}
	host := iq.ToJID().ToBareJID().String()

	ctx.host = host
	ctx.nodeID = nodeID

	// fetch node
	node, err := storage.FetchPubSubNode(host, nodeID)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(iq.InternalServerError())
		return
	}
	if node == nil && opts.failOnNotFound {
		_ = x.router.Route(iq.ItemNotFoundError())
		return
	}
	ctx.node = node

	// fetch affiliations
	var affiliations []pubsubmodel.Affiliation

	if len(opts.allowedAffiliations) > 0 || opts.includeAffiliations {
		affiliations, err = storage.FetchPubSubNodeAffiliations(host, nodeID)
		if err != nil {
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

func checkPresenceAccess(host, jid string) (bool, error) {
	ri, err := storage.FetchRosterItem(host, jid)
	if err != nil {
		return false, err
	}
	allowed := ri != nil && (ri.Subscription == rostermodel.SubscriptionFrom || ri.Subscription == rostermodel.SubscriptionBoth)
	return allowed, nil
}

func checkRosterAccess(host, jid string, allowedGroups []string) (bool, error) {
	ri, err := storage.FetchRosterItem(host, jid)
	if err != nil {
		return false, err
	}
	if ri == nil {
		return false, nil
	}
	for _, group := range ri.Groups {
		for _, allowedGroup := range allowedGroups {
			if group == allowedGroup {
				return true, nil
			}
		}
	}
	return false, nil
}

func checkWhitelistAccess(host, jid string, affiliations []pubsubmodel.Affiliation) bool {
	for _, aff := range affiliations {
		if aff.Affiliation == pubsubmodel.Member {
			return true
		}
	}
	return false
}

func nodeIDRequiredError(stanza xmpp.Stanza) xmpp.Stanza {
	errorElements := []xmpp.XElement{xmpp.NewElementNamespace("nodeid-required", pubSubErrorNamespace)}
	return xmpp.NewErrorStanzaFromStanza(stanza, xmpp.ErrNotAcceptable, errorElements)
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

func subscriptionID(jid, subscription, host, name string) string {
	h := sha256.New()
	h.Write([]byte(jid + subscription + host + name))
	return string(h.Sum(nil))
}
