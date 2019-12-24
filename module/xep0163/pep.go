/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"context"
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

var defaultNodeOptions = pubsubmodel.Options{
	DeliverNotifications:  true,
	DeliverPayloads:       true,
	PersistItems:          true,
	AccessModel:           pubsubmodel.Presence,
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

// Pep represents a Personal Eventing Protocol module.
type Pep struct {
	router      *router.Router
	runQueue    *runqueue.RunQueue
	disco       *xep0030.DiscoInfo
	presenceHub *presencehub.PresenceHub
	hosts       []string
}

// New returns a PEP command IQ handler module.
func New(disco *xep0030.DiscoInfo, presenceHub *presencehub.PresenceHub, router *router.Router) *Pep {
	p := &Pep{
		disco:       disco,
		router:      router,
		runQueue:    runqueue.New("xep0163"),
		presenceHub: presenceHub,
	}
	// register account identity and features
	if disco != nil {
		for _, feature := range pepFeatures {
			disco.RegisterAccountFeature(feature)
		}
	}
	// register disco items
	p.registerDiscoItems()
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
func (x *Pep) ProcessIQ(ctx context.Context, iq *xmpp.IQ) {
	x.runQueue.Run(func() {
		x.processIQ(ctx, iq)
	})
}

// SubscribeToAll subscribes a jid to all host nodes
func (x *Pep) SubscribeToAll(ctx context.Context, host string, jid *jid.JID) {
	x.runQueue.Run(func() {
		if err := x.subscribeToAll(ctx, host, jid); err != nil {
			log.Error(err)
		}
	})
}

// UnsubscribeFromAll unsubscribes a jid from all host nodes
func (x *Pep) UnsubscribeFromAll(ctx context.Context, host string, jid *jid.JID) {
	x.runQueue.Run(func() {
		if err := x.unsubscribeFromAll(ctx, host, jid); err != nil {
			log.Error(err)
		}
	})
}

// DeliverLastItems delivers last items from all those nodes to which the jid is subscribed
func (x *Pep) DeliverLastItems(ctx context.Context, jid *jid.JID) {
	x.runQueue.Run(func() {
		if err := x.deliverLastItems(ctx, jid); err != nil {
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

func (x *Pep) processIQ(ctx context.Context, iq *xmpp.IQ) {
	pubSub := iq.Elements().Child("pubsub")
	switch pubSub.Namespace() {
	case pubSubNamespace:
		x.processRequest(ctx, iq, pubSub)
	case pubSubOwnerNamespace:
		x.processOwnerRequest(ctx, iq, pubSub)
	}
}

func (x *Pep) registerDiscoItems() {
	if x.disco == nil {
		return // nothing to do here
	}
	if err := x.registerDiscoItemHandlers(); err != nil {
		log.Warnf("pep: failed to register disco item handlers: %v", err)
	}
}

func (x *Pep) registerDiscoItemHandlers() error {
	// unregister previous handlers
	for _, h := range x.hosts {
		x.disco.UnregisterProvider(h)
	}
	// register current ones
	hosts, err := storage.FetchHosts()
	if err != nil {
		return err
	}
	for _, host := range hosts {
		x.disco.RegisterProvider(host, &discoInfoProvider{})
	}
	x.hosts = hosts
	return nil
}

func (x *Pep) subscribeToAll(ctx context.Context, host string, subJID *jid.JID) error {
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
			x.notifyOwners(ctx, subscriptionElem, affiliations, host, n.Options.NotificationType)
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
			if err := x.sendLastPublishedItem(ctx, subJID, accessChecker, host, n.Name, n.Options.NotificationType); err != nil {
				return err
			}
		}
	}
	return nil
}

func (x *Pep) unsubscribeFromAll(ctx context.Context, host string, subJID *jid.JID) error {
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

func (x *Pep) deliverLastItems(ctx context.Context, jid *jid.JID) error {
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
		if err := x.sendLastPublishedItem(ctx, jid, accessChecker, node.Host, node.Name, node.Options.NotificationType); err != nil {
			return err
		}
		log.Infof("pep: delivered last item: %s (node: %s, host: %s)", jid.String(), node.Host, node.Name)
	}
	return nil
}

func (x *Pep) processRequest(ctx context.Context, iq *xmpp.IQ, pubSubEl xmpp.XElement) {
	// Create node
	if cmdEl := pubSubEl.Elements().Child("create"); cmdEl != nil && iq.IsSet() {
		x.withCommandContext(ctx, commandOptions{}, cmdEl, iq, func(cmdCtx *commandContext) {
			x.create(ctx, cmdCtx, pubSubEl, iq)
		})
		return
	}
	// Publish
	if cmdEl := pubSubEl.Elements().Child("publish"); cmdEl != nil && iq.IsSet() {
		opts := commandOptions{
			allowedAffiliations:  []string{pubsubmodel.Owner, pubsubmodel.Member},
			includeSubscriptions: true,
		}
		x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
			x.publish(ctx, cmdCtx, cmdEl, iq)
		})
		return
	}
	// Subscribe
	if cmdEl := pubSubEl.Elements().Child("subscribe"); cmdEl != nil && iq.IsSet() {
		opts := commandOptions{
			includeAffiliations: true,
			checkAccess:         true,
			failOnNotFound:      true,
		}
		x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
			x.subscribe(ctx, cmdCtx, cmdEl, iq)
		})
		return
	}
	// Unsubscribe
	if cmdEl := pubSubEl.Elements().Child("unsubscribe"); cmdEl != nil && iq.IsSet() {
		opts := commandOptions{
			includeAffiliations:  true,
			includeSubscriptions: true,
			failOnNotFound:       true,
		}
		x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
			x.unsubscribe(ctx, cmdCtx, cmdEl, iq)
		})
		return
	}
	// Retrieve items
	if cmdEl := pubSubEl.Elements().Child("items"); cmdEl != nil && iq.IsGet() {
		opts := commandOptions{
			includeSubscriptions: true,
			checkAccess:          true,
			failOnNotFound:       true,
		}
		x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
			x.retrieveItems(ctx, cmdCtx, cmdEl, iq)
		})
		return
	}

	_ = x.router.Route(ctx, iq.ServiceUnavailableError())
}

func (x *Pep) processOwnerRequest(ctx context.Context, iq *xmpp.IQ, pubSub xmpp.XElement) {
	// Configure node
	if cmdEl := pubSub.Elements().Child("configure"); cmdEl != nil {
		if iq.IsGet() {
			// send configuration form
			opts := commandOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				failOnNotFound:      true,
			}
			x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
				x.sendConfigurationForm(ctx, cmdCtx, iq)
			})
		} else if iq.IsSet() {
			// update node configuration
			opts := commandOptions{
				allowedAffiliations:  []string{pubsubmodel.Owner},
				includeSubscriptions: true,
				failOnNotFound:       true,
			}
			x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
				x.configure(ctx, cmdCtx, cmdEl, iq)
			})
		} else {
			_ = x.router.Route(ctx, iq.ServiceUnavailableError())
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
			x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
				x.retrieveAffiliations(ctx, cmdCtx, iq)
			})
		} else if iq.IsSet() {
			opts := commandOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				failOnNotFound:      true,
			}
			x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
				x.updateAffiliations(ctx, cmdCtx, cmdEl, iq)
			})
		} else {
			_ = x.router.Route(ctx, iq.ServiceUnavailableError())
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
			x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
				x.retrieveSubscriptions(ctx, cmdCtx, iq)
			})
		} else if iq.IsSet() {
			opts := commandOptions{
				allowedAffiliations: []string{pubsubmodel.Owner},
				failOnNotFound:      true,
			}
			x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
				x.updateSubscriptions(ctx, cmdCtx, cmdEl, iq)
			})
		} else {
			_ = x.router.Route(ctx, iq.ServiceUnavailableError())
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
		x.withCommandContext(ctx, opts, cmdEl, iq, func(cmdCtx *commandContext) {
			x.delete(ctx, cmdCtx, iq)
		})
		return
	}

	_ = x.router.Route(ctx, iq.FeatureNotImplementedError())
}

func (x *Pep) create(ctx context.Context, cmdCtx *commandContext, pubSubEl xmpp.XElement, iq *xmpp.IQ) {
	if cmdCtx.node != nil {
		_ = x.router.Route(ctx, iq.ConflictError())
		return
	}
	node := &pubsubmodel.Node{
		Host: cmdCtx.host,
		Name: cmdCtx.nodeID,
	}
	if configEl := pubSubEl.Elements().Child("configure"); configEl != nil {
		form, err := xep0004.NewFormFromElement(configEl)
		if err != nil {
			_ = x.router.Route(ctx, iq.BadRequestError())
			return
		}
		opts, err := pubsubmodel.NewOptionsFromSubmitForm(form)
		if err != nil {
			_ = x.router.Route(ctx, iq.BadRequestError())
			return
		}
		node.Options = *opts
	} else {
		// apply default configuration
		node.Options = defaultNodeOptions
	}
	if err := x.createNode(ctx, node); err != nil {
		log.Error(err)
		_ = x.router.Route(ctx, iq.InternalServerError())
		return
	}
	log.Infof("pep: created node (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(ctx, iq.ResultIQ())
}

func (x *Pep) sendConfigurationForm(ctx context.Context, cmdCtx *commandContext, iq *xmpp.IQ) {
	// compose config form response
	configureNode := xmpp.NewElementName("configure")
	configureNode.SetAttribute("node", cmdCtx.nodeID)

	rosterGroups, err := storage.FetchRosterGroups(iq.ToJID().Node())
	if err != nil {
		log.Error(err)
		_ = x.router.Route(ctx, iq.InternalServerError())
		return
	}

	configureNode.AppendElement(cmdCtx.node.Options.Form(rosterGroups).Element())

	pubSubNode := xmpp.NewElementNamespace("pubsub", pubSubOwnerNamespace)
	pubSubNode.AppendElement(configureNode)

	res := iq.ResultIQ()
	res.AppendElement(pubSubNode)

	log.Infof("pep: sent configuration form (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(ctx, res)
}

func (x *Pep) configure(ctx context.Context, cmdCtx *commandContext, cmdElem xmpp.XElement, iq *xmpp.IQ) {
	formEl := cmdElem.Elements().ChildNamespace("x", xep0004.FormNamespace)
	if formEl == nil {
		_ = x.router.Route(ctx, iq.NotAcceptableError())
		return
	}
	configForm, err := xep0004.NewFormFromElement(formEl)
	if err != nil {
		_ = x.router.Route(ctx, iq.NotAcceptableError())
		return
	}
	nodeOpts, err := pubsubmodel.NewOptionsFromSubmitForm(configForm)
	if err != nil {
		_ = x.router.Route(ctx, iq.NotAcceptableError())
		return
	}
	cmdCtx.node.Options = *nodeOpts

	// update node config
	if err := storage.UpsertNode(cmdCtx.node); err != nil {
		log.Error(err)
		_ = x.router.Route(ctx, iq.InternalServerError())
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
			ctx,
			configElem,
			cmdCtx.subscriptions,
			cmdCtx.accessChecker,
			cmdCtx.host,
			cmdCtx.nodeID,
			opts.NotificationType)
	}
	log.Infof("pep: node configuration updated (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(ctx, iq.ResultIQ())
}

func (x *Pep) delete(ctx context.Context, cmdCtx *commandContext, iq *xmpp.IQ) {
	// delete node
	if err := storage.DeleteNode(cmdCtx.host, cmdCtx.nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(ctx, iq.InternalServerError())
		return
	}
	// notify delete
	opts := cmdCtx.node.Options

	if opts.DeliverNotifications && opts.NotifyDelete {
		deleteElem := xmpp.NewElementName("delete")
		deleteElem.SetAttribute("node", cmdCtx.nodeID)

		x.notifySubscribers(
			ctx,
			deleteElem,
			cmdCtx.subscriptions,
			cmdCtx.accessChecker,
			cmdCtx.host,
			cmdCtx.nodeID,
			opts.NotificationType)
	}
	log.Infof("pep: deleted node (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	x.registerDiscoItems()
	_ = x.router.Route(ctx, iq.ResultIQ())
}

func (x *Pep) subscribe(ctx context.Context, cmdCtx *commandContext, cmdEl xmpp.XElement, iq *xmpp.IQ) {
	// validate JID portion
	subJID := cmdEl.Attributes().Get("jid")
	if subJID != iq.FromJID().ToBareJID().String() {
		_ = x.router.Route(ctx, invalidJIDError(iq))
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
		_ = x.router.Route(ctx, iq.InternalServerError())
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
		x.notifyOwners(ctx, subscriptionElem, cmdCtx.affiliations, cmdCtx.host, opts.NotificationType)
	}
	// send last node item
	switch opts.SendLastPublishedItem {
	case pubsubmodel.OnSub, pubsubmodel.OnSubAndPresence:
		subscriberJID, _ := jid.NewWithString(sub.JID, true)
		err := x.sendLastPublishedItem(ctx, subscriberJID, cmdCtx.accessChecker, cmdCtx.host, cmdCtx.nodeID, opts.NotificationType)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(ctx, iq.InternalServerError())
			return
		}
	}

	// compose response
	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	pubSubElem.AppendElement(subscriptionElem)
	iqRes.AppendElement(pubSubElem)

	_ = x.router.Route(ctx, iqRes)
}

func (x *Pep) unsubscribe(ctx context.Context, cmdCtx *commandContext, cmdEl xmpp.XElement, iq *xmpp.IQ) {
	subJID := cmdEl.Attributes().Get("jid")
	if subJID != iq.FromJID().ToBareJID().String() {
		_ = x.router.Route(ctx, iq.ForbiddenError())
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
		_ = x.router.Route(ctx, notSubscribedError(iq))
		return
	}
	// delete subscription
	if err := storage.DeleteNodeSubscription(subJID, cmdCtx.host, cmdCtx.nodeID); err != nil {
		log.Error(err)
		_ = x.router.Route(ctx, iq.InternalServerError())
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
		x.notifyOwners(ctx, subscriptionElem, cmdCtx.affiliations, cmdCtx.host, opts.NotificationType)
	}

	// compose response
	iqRes := iq.ResultIQ()
	pubSubElem := xmpp.NewElementNamespace("pubsub", pubSubNamespace)
	pubSubElem.AppendElement(subscriptionElem)
	iqRes.AppendElement(pubSubElem)

	_ = x.router.Route(ctx, iqRes)
}

func (x *Pep) publish(ctx context.Context, cmdCtx *commandContext, cmdEl xmpp.XElement, iq *xmpp.IQ) {
	itemEl := cmdEl.Elements().Child("item")
	if itemEl == nil || len(itemEl.Elements().All()) != 1 {
		_ = x.router.Route(ctx, invalidPayloadError(iq))
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
			_ = x.router.Route(ctx, iq.ForbiddenError())
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
		if err := x.createNode(ctx, cmdCtx.node); err != nil {
			log.Error(err)
			_ = x.router.Route(ctx, iq.InternalServerError())
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
			_ = x.router.Route(ctx, iq.InternalServerError())
			return
		}
	}
	log.Infof("pep: published item (host: %s, node_id: %s, item_id: %s)", cmdCtx.host, cmdCtx.nodeID, itemID)

	// notify published item
	itemsElem := xmpp.NewElementName("items")
	itemsElem.SetAttribute("node", cmdCtx.nodeID)

	itemElem := xmpp.NewElementName("item")
	itemElem.SetAttribute("id", itemID)
	if opts.DeliverPayloads || !opts.PersistItems {
		itemElem.AppendElement(itemEl.Elements().All()[0])
	}
	itemsElem.AppendElement(itemElem)

	x.notifySubscribers(
		ctx,
		itemsElem,
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

	_ = x.router.Route(ctx, iqRes)
}

func (x *Pep) retrieveItems(ctx context.Context, cmdCtx *commandContext, cmdEl xmpp.XElement, iq *xmpp.IQ) {
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
		_ = x.router.Route(ctx, iq.InternalServerError())
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

	_ = x.router.Route(ctx, iqRes)
}

func (x *Pep) retrieveAffiliations(ctx context.Context, cmdCtx *commandContext, iq *xmpp.IQ) {
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

	_ = x.router.Route(ctx, iqRes)
}

func (x *Pep) updateAffiliations(ctx context.Context, cmdCtx *commandContext, cmdElem xmpp.XElement, iq *xmpp.IQ) {
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
			_ = x.router.Route(ctx, iq.BadRequestError())
			return
		}
		if err != nil {
			log.Error(err)
			_ = x.router.Route(ctx, iq.InternalServerError())
			return
		}
	}
	log.Infof("pep: modified affiliations (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(ctx, iq.ResultIQ())
}

func (x *Pep) retrieveSubscriptions(ctx context.Context, cmdCtx *commandContext, iq *xmpp.IQ) {
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

	_ = x.router.Route(ctx, iqRes)
}

func (x *Pep) updateSubscriptions(ctx context.Context, cmdCtx *commandContext, cmdElem xmpp.XElement, iq *xmpp.IQ) {
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
			_ = x.router.Route(ctx, iq.BadRequestError())
			return
		}
		if err != nil {
			log.Error(err)
			_ = x.router.Route(ctx, iq.InternalServerError())
			return
		}
	}
	log.Infof("pep: modified subscriptions (host: %s, node_id: %s)", cmdCtx.host, cmdCtx.nodeID)

	_ = x.router.Route(ctx, iq.ResultIQ())
}

func (x *Pep) notifyOwners(ctx context.Context, notificationElem xmpp.XElement, affiliations []pubsubmodel.Affiliation, host, notificationType string) {
	hostJID, _ := jid.NewWithString(host, true)
	for _, affiliation := range affiliations {
		if affiliation.Affiliation != pubsubmodel.Owner {
			continue
		}
		toJID, _ := jid.NewWithString(affiliation.JID, true)
		eventMsg := eventMessage(notificationElem, hostJID, toJID, notificationType)

		_ = x.router.Route(ctx, eventMsg)
	}
}

func (x *Pep) notifySubscribers(
	ctx context.Context,
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
	x.notify(ctx, notificationElem, toJIDs, accessChecker, host, nodeID, notificationType)
}

func (x *Pep) notify(
	ctx context.Context,
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
			err := accessChecker.checkAccess(toJID.ToBareJID().String())
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
					goto broadcastEventMsg // broadcast when caps are pending to be fetched
				}
				if !caps.HasFeature(nodeID + "+notify") {
					continue
				}
				// notify to full jid
				presence := onlinePresence.Presence

				eventMsg := eventMessage(notificationElem, hostJID, presence.FromJID(), notificationType)
				_ = x.router.Route(ctx, eventMsg)
			}
			return
		}
	broadcastEventMsg:
		// broadcast event message
		eventMsg := eventMessage(notificationElem, hostJID, &toJID, notificationType)
		_ = x.router.Route(ctx, eventMsg)
	}
}

func (x *Pep) withCommandContext(ctx context.Context, opts commandOptions, cmdElem xmpp.XElement, iq *xmpp.IQ, fn func(cmdCtx *commandContext)) {
	var cmdCtx commandContext

	nodeID := cmdElem.Attributes().Get("node")
	if len(nodeID) == 0 {
		_ = x.router.Route(ctx, nodeIDRequiredError(iq))
		return
	}
	fromJID := iq.FromJID().ToBareJID().String()
	host := iq.ToJID().ToBareJID().String()

	cmdCtx.host = host
	cmdCtx.nodeID = nodeID
	cmdCtx.isAccountOwner = fromJID == host

	// fetch node
	node, err := storage.FetchNode(host, nodeID)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(ctx, iq.InternalServerError())
		return
	}
	if node == nil {
		if opts.failOnNotFound {
			_ = x.router.Route(ctx, iq.ItemNotFoundError())
		} else {
			fn(&cmdCtx)
		}
		return
	}
	cmdCtx.node = node

	// fetch affiliation
	aff, err := storage.FetchNodeAffiliation(host, nodeID, fromJID)
	if err != nil {
		log.Error(err)
		_ = x.router.Route(ctx, iq.InternalServerError())
		return
	}
	cmdCtx.accessChecker = &accessChecker{
		host:                node.Host,
		nodeID:              node.Name,
		accessModel:         node.Options.AccessModel,
		rosterAllowedGroups: node.Options.RosterGroupsAllowed,
		affiliation:         aff,
	}
	// check access
	if opts.checkAccess && !cmdCtx.isAccountOwner {
		err := cmdCtx.accessChecker.checkAccess(fromJID)
		switch err {
		case nil:
			break

		case errOutcastMember:
			_ = x.router.Route(ctx, iq.ForbiddenError())
			return

		case errPresenceSubscriptionRequired:
			_ = x.router.Route(ctx, presenceSubscriptionRequiredError(iq))
			return

		case errNotInRosterGroup:
			_ = x.router.Route(ctx, notInRosterGroupError(iq))
			return

		case errNotOnWhiteList:
			_ = x.router.Route(ctx, notOnWhitelistError(iq))
			return

		default:
			log.Error(err)
			_ = x.router.Route(ctx, iq.InternalServerError())
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
			_ = x.router.Route(ctx, iq.ForbiddenError())
			return
		}
	}
	// fetch subscriptions
	if opts.includeSubscriptions {
		subscriptions, err := storage.FetchNodeSubscriptions(host, nodeID)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(ctx, iq.InternalServerError())
			return
		}
		cmdCtx.subscriptions = subscriptions
	}
	// fetch affiliations
	if opts.includeAffiliations {
		affiliations, err := storage.FetchNodeAffiliations(host, nodeID)
		if err != nil {
			log.Error(err)
			_ = x.router.Route(ctx, iq.InternalServerError())
			return
		}
		cmdCtx.affiliations = affiliations
	}
	fn(&cmdCtx)
}

func (x *Pep) createNode(ctx context.Context, node *pubsubmodel.Node) error {
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
	if err := storage.UpsertNodeSubscription(ownerSub, node.Host, node.Name); err != nil {
		return err
	}
	x.registerDiscoItems()
	return nil
}

func (x *Pep) sendLastPublishedItem(ctx context.Context, toJID *jid.JID, accessChecker *accessChecker, host, nodeID, notificationType string) error {
	node, err := storage.FetchNode(host, nodeID)
	if err != nil {
		return err
	}
	if node == nil {
		return nil
	}
	lastItem, err := storage.FetchNodeLastItem(host, nodeID)
	if err != nil {
		return err
	}
	if lastItem == nil {
		return nil
	}
	itemsEl := xmpp.NewElementName("items")
	itemsEl.SetAttribute("node", nodeID)
	itemEl := xmpp.NewElementName("item")
	itemEl.SetAttribute("id", lastItem.ID)
	if node.Options.DeliverPayloads || !node.Options.PersistItems {
		itemEl.AppendElement(lastItem.Payload)
	}
	itemsEl.AppendElement(itemEl)

	x.notify(
		ctx,
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
