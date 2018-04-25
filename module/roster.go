/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const rosterNamespace = "jabber:iq:roster"

const (
	rosterRequestedContextKey = "roster:requested"
)

// ModRoster represents a roster server stream module.
type ModRoster struct {
	cfg        *config.ModRoster
	stm        c2s.Stream
	actorCh    chan func()
	errHandler func(error)
}

// NewRoster returns a roster server stream module.
func NewRoster(cfg *config.ModRoster, stm c2s.Stream) *ModRoster {
	r := &ModRoster{
		cfg:        cfg,
		stm:        stm,
		actorCh:    make(chan func(), moduleMailboxSize),
		errHandler: func(err error) { log.Error(err) },
	}
	go r.actorLoop()
	return r
}

// AssociatedNamespaces returns namespaces associated
// with roster module.
func (r *ModRoster) AssociatedNamespaces() []string {
	return []string{}
}

// Done signals stream termination.
func (r *ModRoster) Done() {
}

// MatchesIQ returns whether or not an IQ should be
// processed by the roster module.
func (r *ModRoster) MatchesIQ(iq *xml.IQ) bool {
	return iq.Elements().ChildNamespace("query", rosterNamespace) != nil
}

// ProcessIQ processes a roster IQ taking according actions
// over the associated stream.
func (r *ModRoster) ProcessIQ(iq *xml.IQ) {
	r.actorCh <- func() {
		q := iq.Elements().ChildNamespace("query", rosterNamespace)
		if iq.IsGet() {
			r.sendRoster(iq, q)
		} else if iq.IsSet() {
			r.updateRoster(iq, q)
		} else {
			r.stm.SendElement(iq.BadRequestError())
		}
	}
}

// ProcessPresence process an incoming roster presence.
func (r *ModRoster) ProcessPresence(presence *xml.Presence) {
	r.actorCh <- func() {
		if err := r.processPresence(presence); err != nil {
			r.errHandler(err)
		}
	}
}

// DeliverPendingApprovalNotifications delivers any pending roster notification
// to the associated stream.
func (r *ModRoster) DeliverPendingApprovalNotifications() {
	r.actorCh <- func() {
		if err := r.deliverPendingApprovalNotifications(); err != nil {
			r.errHandler(err)
		}
	}
}

// ReceivePresences delivers all inbound roster available presences
// to the associated module stream.
func (r *ModRoster) ReceivePresences() {
	r.actorCh <- func() {
		if err := r.receivePresences(); err != nil {
			r.errHandler(err)
		}
	}
}

// BroadcastPresence broadcasts presence to all outbound roster contacts.
func (r *ModRoster) BroadcastPresence(presence *xml.Presence) {
	r.actorCh <- func() {
		if err := r.broadcastPresence(presence); err != nil {
			r.errHandler(err)
		}
	}
}

// BroadcastPresenceAndWait broadcasts presence to all outbound
// roster contacts in a synchronous manner.
func (r *ModRoster) BroadcastPresenceAndWait(presence *xml.Presence) {
	continueCh := make(chan struct{})
	r.actorCh <- func() {
		if err := r.broadcastPresence(presence); err != nil {
			r.errHandler(err)
		}
		close(continueCh)
	}
	<-continueCh
}

func (r *ModRoster) actorLoop() {
	for {
		select {
		case f := <-r.actorCh:
			f()
		}
	}
}

func (r *ModRoster) processPresence(presence *xml.Presence) error {
	switch presence.Type() {
	case xml.SubscribeType:
		return r.processSubscribe(presence)
	case xml.SubscribedType:
		return r.processSubscribed(presence)
	case xml.UnsubscribeType:
		return r.processUnsubscribe(presence)
	case xml.UnsubscribedType:
		return r.processUnsubscribed(presence)
	}
	return nil
}

func (r *ModRoster) deliverPendingApprovalNotifications() error {
	rosterNotifications, err := storage.Instance().FetchRosterNotifications(r.stm.Username())
	if err != nil {
		return err
	}
	for _, rosterNotification := range rosterNotifications {
		fromJID, _ := xml.NewJID(rosterNotification.User, r.stm.Domain(), "", true)
		p := xml.NewPresence(fromJID, r.stm.JID(), xml.SubscribeType)
		p.AppendElements(rosterNotification.Elements)
		r.stm.SendElement(p)
	}
	return nil
}

func (r *ModRoster) receivePresences() error {
	items, _, err := storage.Instance().FetchRosterItems(r.stm.JID().Node())
	if err != nil {
		return err
	}
	userJID := r.stm.JID()
	for _, item := range items {
		switch item.Subscription {
		case subscriptionTo, subscriptionBoth:
			r.routePresencesFrom(r.rosterItemJID(&item), userJID, xml.AvailableType)
		}
	}
	return nil
}

func (r *ModRoster) broadcastPresence(presence *xml.Presence) error {
	items, _, err := storage.Instance().FetchRosterItems(r.stm.JID().Node())
	if err != nil {
		return err
	}
	for _, item := range items {
		switch item.Subscription {
		case subscriptionFrom, subscriptionBoth:
			r.routePresence(presence, r.rosterItemJID(&item))
		}
	}
	return nil
}

func (r *ModRoster) sendRoster(iq *xml.IQ, query xml.XElement) {
	if query.Elements().Count() > 0 {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	log.Infof("retrieving user roster... (%s/%s)", r.stm.Username(), r.stm.Resource())

	items, ver, err := storage.Instance().FetchRosterItems(r.stm.JID().Node())
	if err != nil {
		r.errHandler(err)
		r.stm.SendElement(iq.InternalServerError())
		return
	}
	v := r.parseVer(query.Attributes().Get("ver"))

	result := iq.ResultIQ()
	if !r.cfg.Versioning || v == 0 || v < ver.DeletionVer {
		// push all roster items
		q := xml.NewElementNamespace("query", rosterNamespace)
		if r.cfg.Versioning {
			q.SetAttribute("ver", fmt.Sprintf("v%d", ver.Ver))
		}
		for _, item := range items {
			q.AppendElement(r.elementFromRosterItem(&item))
		}
		result.AppendElement(q)
		r.stm.SendElement(result)
	} else {
		// push roster changes
		r.stm.SendElement(result)
		for _, item := range items {
			if item.Ver > v {
				iq := xml.NewIQType(uuid.New(), xml.SetType)
				q := xml.NewElementNamespace("query", rosterNamespace)
				q.SetAttribute("ver", fmt.Sprintf("v%d", item.Ver))
				q.AppendElement(r.elementFromRosterItem(&item))
				iq.AppendElement(q)
				r.stm.SendElement(iq)
			}
		}
	}
	r.stm.Context().SetBool(true, rosterRequestedContextKey)
}

func (r *ModRoster) updateRoster(iq *xml.IQ, query xml.XElement) {
	items := query.Elements().Children("item")
	if len(items) != 1 {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	ri, err := r.rosterItemFromElement(items[0])
	if err != nil {
		r.stm.SendElement(iq.BadRequestError())
		return
	}
	switch ri.Subscription {
	case subscriptionRemove:
		if err := r.removeItem(ri); err != nil {
			r.errHandler(err)
			r.stm.SendElement(iq.InternalServerError())
			return
		}
	default:
		if err := r.updateItem(ri); err != nil {
			r.errHandler(err)
			r.stm.SendElement(iq.InternalServerError())
			return
		}
	}
	r.stm.SendElement(iq.ResultIQ())
}

func (r *ModRoster) removeItem(ri *model.RosterItem) error {
	usrJID := r.stm.JID()
	cntJID := r.rosterItemJID(ri)

	log.Infof("removing roster item: %v (%s/%s)", cntJID, r.stm.Username(), r.stm.Resource())

	var unsubscribe *xml.Presence
	var unsubscribed *xml.Presence

	usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.Node())
	if err != nil {
		return err
	}
	userSubscription := subscriptionNone
	if usrRi != nil {
		userSubscription = usrRi.Subscription
		switch userSubscription {
		case subscriptionTo:
			unsubscribe = xml.NewPresence(usrJID.ToBareJID(), cntJID.ToBareJID(), xml.UnsubscribeType)
		case subscriptionFrom:
			unsubscribed = xml.NewPresence(usrJID.ToBareJID(), cntJID.ToBareJID(), xml.UnsubscribedType)
		case subscriptionBoth:
			unsubscribe = xml.NewPresence(usrJID.ToBareJID(), cntJID.ToBareJID(), xml.UnsubscribeType)
			unsubscribed = xml.NewPresence(usrJID.ToBareJID(), cntJID.ToBareJID(), xml.UnsubscribedType)
		}
		usrRi.Subscription = subscriptionRemove
		usrRi.Ask = false

		if err := r.deleteNotification(usrJID, cntJID); err != nil {
			return err
		}
		if err := r.deleteItem(usrRi, usrJID); err != nil {
			return err
		}
	}

	if c2s.Instance().IsLocalDomain(cntJID.Domain()) {
		cntRi, err := storage.Instance().FetchRosterItem(cntJID.Node(), usrJID.Node())
		if err != nil {
			return err
		}
		if cntRi != nil {
			if cntRi.Subscription == subscriptionFrom || cntRi.Subscription == subscriptionBoth {
				r.routePresencesFrom(cntJID, usrJID, xml.UnavailableType)
			}
			switch cntRi.Subscription {
			case subscriptionBoth:
				cntRi.Subscription = subscriptionTo
				if r.insertOrUpdateItem(cntRi, cntJID); err != nil {
					return err
				}
				fallthrough

			default:
				cntRi.Subscription = subscriptionNone
				if r.insertOrUpdateItem(cntRi, cntJID); err != nil {
					return err
				}
			}
		}
	}
	if unsubscribe != nil {
		r.routePresence(unsubscribe, cntJID)
	}
	if unsubscribed != nil {
		r.routePresence(unsubscribed, cntJID)
	}
	if userSubscription == subscriptionFrom || userSubscription == subscriptionBoth {
		r.routePresencesFrom(usrJID, cntJID, xml.UnavailableType)
	}
	return nil
}

func (r *ModRoster) updateItem(ri *model.RosterItem) error {
	usrJID := r.stm.JID()
	cntJID := r.rosterItemJID(ri)

	log.Infof("updating roster item - contact: %s (%s/%s)", cntJID, r.stm.Username(), r.stm.Resource())

	usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.Node())
	if err != nil {
		return err
	}
	if usrRi != nil {
		// update roster item
		if len(ri.Name) > 0 {
			usrRi.Name = ri.Name
		}
		usrRi.Groups = ri.Groups

	} else {
		usrRi = &model.RosterItem{
			User:         r.stm.Username(),
			Contact:      ri.Contact,
			Name:         ri.Name,
			Subscription: subscriptionNone,
			Groups:       ri.Groups,
			Ask:          ri.Ask,
		}
	}
	return r.insertOrUpdateItem(usrRi, r.stm.JID())
}

func (r *ModRoster) processSubscribe(presence *xml.Presence) error {
	usrJID := r.stm.JID()
	cntJID := presence.ToJID()

	log.Infof("processing 'subscribe' - contact: %s (%s/%s)", cntJID, r.stm.Username(), r.stm.Resource())

	usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.Node())
	if err != nil {
		return err
	}
	if usrRi != nil {
		switch usrRi.Subscription {
		case subscriptionTo, subscriptionBoth:
			return nil // already subscribed...
		default:
			if !usrRi.Ask {
				usrRi.Ask = true
			} else {
				return nil // notification already sent...
			}
		}
	} else {
		// create roster item if not previously created
		usrRi = &model.RosterItem{
			User:         usrJID.Node(),
			Contact:      cntJID.Node(),
			Subscription: subscriptionNone,
			Ask:          true,
		}
	}
	if r.insertOrUpdateItem(usrRi, usrJID); err != nil {
		return err
	}
	// stamp the presence stanza of type "subscribe" with the user's bare JID as the 'from' address
	p := xml.NewPresence(usrJID.ToBareJID(), cntJID.ToBareJID(), xml.SubscribeType)
	p.AppendElements(presence.Elements().All())

	if c2s.Instance().IsLocalDomain(cntJID.Domain()) {
		// archive roster approval notification
		if err := r.insertOrUpdateNotification(usrJID, cntJID, p); err != nil {
			return err
		}
	}
	r.routePresence(p, cntJID)
	return nil
}

func (r *ModRoster) processSubscribed(presence *xml.Presence) error {
	usrJID := presence.ToJID()
	cntJID := r.stm.JID()

	log.Infof("processing 'subscribed' - user: %s (%s/%s)", usrJID, r.stm.Username(), r.stm.Resource())

	if err := r.deleteNotification(usrJID, cntJID); err != nil {
		return err
	}
	cntRi, err := storage.Instance().FetchRosterItem(cntJID.Node(), usrJID.Node())
	if err != nil {
		return err
	}
	if cntRi != nil {
		switch cntRi.Subscription {
		case subscriptionTo:
			cntRi.Subscription = subscriptionBoth
		case subscriptionNone:
			cntRi.Subscription = subscriptionFrom
		}
	} else {
		// create roster item if not previously created
		cntRi = &model.RosterItem{
			User:         cntJID.Node(),
			Contact:      usrJID.Node(),
			Subscription: subscriptionFrom,
			Ask:          false,
		}
	}
	if r.insertOrUpdateItem(cntRi, cntJID); err != nil {
		return err
	}
	// stamp the presence stanza of type "subscribed" with the contact's bare JID as the 'from' address
	p := xml.NewPresence(cntJID.ToBareJID(), usrJID.ToBareJID(), xml.SubscribedType)
	p.AppendElements(presence.Elements().All())

	if c2s.Instance().IsLocalDomain(usrJID.Domain()) {
		usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.Node())
		if err != nil {
			return err
		}
		if usrRi != nil {
			switch usrRi.Subscription {
			case subscriptionFrom:
				usrRi.Subscription = subscriptionBoth
			case subscriptionNone:
				usrRi.Subscription = subscriptionTo
			default:
				return nil
			}
			usrRi.Ask = false
			if r.insertOrUpdateItem(usrRi, usrJID); err != nil {
				return err
			}
		}
	}
	r.routePresence(p, usrJID)
	r.routePresencesFrom(cntJID, usrJID, xml.AvailableType)
	return nil
}

func (r *ModRoster) processUnsubscribe(presence *xml.Presence) error {
	usrJID := r.stm.JID()
	cntJID := presence.ToJID()

	log.Infof("processing 'unsubscribe' - contact: %s (%s/%s)", cntJID, r.stm.Username(), r.stm.Resource())

	usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.Node())
	if err != nil {
		return err
	}
	usrSub := subscriptionNone
	if usrRi != nil {
		usrSub = usrRi.Subscription
		switch usrSub {
		case subscriptionBoth:
			usrRi.Subscription = subscriptionFrom
		default:
			usrRi.Subscription = subscriptionNone
		}
		if r.insertOrUpdateItem(usrRi, usrJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "unsubscribe" with the users's bare JID as the 'from' address
	p := xml.NewPresence(usrJID.ToBareJID(), cntJID.ToBareJID(), xml.UnsubscribeType)
	p.AppendElements(presence.Elements().All())

	if c2s.Instance().IsLocalDomain(cntJID.Domain()) {
		cntRi, err := storage.Instance().FetchRosterItem(cntJID.Node(), usrJID.Node())
		if err != nil {
			return err
		}
		if cntRi != nil {
			switch cntRi.Subscription {
			case subscriptionBoth:
				cntRi.Subscription = subscriptionTo
			default:
				cntRi.Subscription = subscriptionNone
			}
			if r.insertOrUpdateItem(cntRi, cntJID); err != nil {
				return err
			}
		}
	}
	r.routePresence(p, cntJID)

	if usrSub == subscriptionTo || usrSub == subscriptionBoth {
		r.routePresencesFrom(cntJID, usrJID, xml.UnavailableType)
	}
	return nil
}

func (r *ModRoster) processUnsubscribed(presence *xml.Presence) error {
	usrJID := presence.ToJID()
	cntJID := r.stm.JID()

	log.Infof("processing 'unsubscribed' - user: %s (%s/%s)", usrJID, r.stm.Username(), r.stm.Resource())

	if err := r.deleteNotification(usrJID, cntJID); err != nil {
		return err
	}
	cntRi, err := storage.Instance().FetchRosterItem(cntJID.Node(), usrJID.Node())
	if err != nil {
		return err
	}
	cntSub := subscriptionNone
	if cntRi != nil {
		cntSub = cntRi.Subscription
		switch cntSub {
		case subscriptionBoth:
			cntRi.Subscription = subscriptionTo
		default:
			cntRi.Subscription = subscriptionNone
		}
		if r.insertOrUpdateItem(cntRi, cntJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "unsubscribed" with the contact's bare JID as the 'from' address
	p := xml.NewPresence(cntJID.ToBareJID(), usrJID.ToBareJID(), xml.UnsubscribedType)
	p.AppendElements(presence.Elements().All())

	if c2s.Instance().IsLocalDomain(usrJID.Domain()) {
		usrRi, err := storage.Instance().FetchRosterItem(usrJID.Node(), cntJID.Node())
		if err != nil {
			return err
		}
		if usrRi != nil {
			switch usrRi.Subscription {
			case subscriptionBoth:
				usrRi.Subscription = subscriptionFrom
			default:
				usrRi.Subscription = subscriptionNone
			}
			usrRi.Ask = false
			if r.insertOrUpdateItem(usrRi, usrJID); err != nil {
				return err
			}
		}
	}
	r.routePresence(p, usrJID)

	if cntSub == subscriptionFrom || cntSub == subscriptionBoth {
		r.routePresencesFrom(cntJID, usrJID, xml.UnavailableType)
	}
	return nil
}

func (r *ModRoster) insertOrUpdateNotification(usrJID *xml.JID, cntJID *xml.JID, presence *xml.Presence) error {
	rn := &model.RosterNotification{
		User:     usrJID.Node(),
		Contact:  cntJID.Node(),
		Elements: presence.Elements().All(),
	}
	return storage.Instance().InsertOrUpdateRosterNotification(rn)
}

func (r *ModRoster) deleteNotification(usrJID *xml.JID, cntJID *xml.JID) error {
	return storage.Instance().DeleteRosterNotification(usrJID.Node(), cntJID.Node())
}

func (r *ModRoster) insertOrUpdateItem(ri *model.RosterItem, pushTo *xml.JID) error {
	v, err := storage.Instance().InsertOrUpdateRosterItem(ri)
	if err != nil {
		return err
	}
	ri.Ver = v.Ver
	return r.pushItem(ri, pushTo)
}

func (r *ModRoster) deleteItem(ri *model.RosterItem, pushTo *xml.JID) error {
	v, err := storage.Instance().DeleteRosterItem(ri.User, ri.Contact)
	if err != nil {
		return err
	}
	ri.Ver = v.Ver
	return r.pushItem(ri, pushTo)
}

func (r *ModRoster) pushItem(ri *model.RosterItem, to *xml.JID) error {
	query := xml.NewElementNamespace("query", rosterNamespace)
	if r.cfg.Versioning {
		query.SetAttribute("ver", fmt.Sprintf("v%d", ri.Ver))
	}
	query.AppendElement(r.elementFromRosterItem(ri))

	stms := c2s.Instance().AvailableStreams(to.Node())
	for _, stm := range stms {
		if !stm.Context().Bool(rosterRequestedContextKey) {
			continue
		}
		pushEl := xml.NewIQType(uuid.New(), xml.SetType)
		pushEl.SetTo(stm.JID().String())
		pushEl.AppendElement(query)
		stm.SendElement(pushEl)
	}
	return nil
}

func (r *ModRoster) routePresencesFrom(from *xml.JID, to *xml.JID, presenceType string) {
	fromStreams := c2s.Instance().AvailableStreams(from.Node())
	for _, fromStream := range fromStreams {
		p := xml.NewPresence(fromStream.JID(), to.ToBareJID(), presenceType)
		if presence := fromStream.Presence(); presence != nil && presenceType == xml.AvailableType {
			p.AppendElements(presence.Elements().All())
		}
		r.routePresence(p, to)
	}
}

func (r *ModRoster) routePresence(presence *xml.Presence, to *xml.JID) {
	if c2s.Instance().IsLocalDomain(to.Domain()) {
		toStreams := c2s.Instance().AvailableStreams(to.Node())
		for _, toStream := range toStreams {
			p := xml.NewPresence(presence.FromJID(), toStream.JID(), presence.Type())
			p.AppendElements(presence.Elements().All())
			toStream.SendElement(p)
		}
	} else {
		// TODO(ortuman): Implement XMPP federation
	}
}

func (r *ModRoster) rosterItemJID(ri *model.RosterItem) *xml.JID {
	j, _ := xml.NewJIDString(fmt.Sprintf("%s@%s", ri.Contact, r.stm.Domain()), true)
	return j
}

func (r *ModRoster) rosterItemFromElement(item xml.XElement) (*model.RosterItem, error) {
	ri := &model.RosterItem{}
	if jid := item.Attributes().Get("jid"); len(jid) > 0 {
		j, err := xml.NewJIDString(jid, false)
		if err != nil {
			return nil, err
		}
		ri.Contact = j.Node()
	} else {
		return nil, errors.New("item 'jid' attribute is required")
	}
	ri.Name = item.Attributes().Get("name")

	subscription := item.Attributes().Get("subscription")
	if len(subscription) > 0 {
		switch subscription {
		case subscriptionBoth, subscriptionFrom, subscriptionTo, subscriptionNone, subscriptionRemove:
			break
		default:
			return nil, fmt.Errorf("unrecognized 'subscription' enum type: %s", subscription)
		}
		ri.Subscription = subscription
	}
	ask := item.Attributes().Get("ask")
	if len(ask) > 0 {
		if ask != "subscribe" {
			return nil, fmt.Errorf("unrecognized 'ask' enum type: %s", subscription)
		}
		ri.Ask = true
	}
	groups := item.Elements().Children("group")
	for _, group := range groups {
		if group.Attributes().Count() > 0 {
			return nil, errors.New("group element must not contain any attribute")
		}
		ri.Groups = append(ri.Groups, group.Text())
	}
	return ri, nil
}

func (r *ModRoster) elementFromRosterItem(ri *model.RosterItem) xml.XElement {
	riJID := r.rosterItemJID(ri)
	item := xml.NewElementName("item")
	item.SetAttribute("jid", riJID.ToBareJID().String())
	if len(ri.Name) > 0 {
		item.SetAttribute("name", ri.Name)
	}
	if len(ri.Subscription) > 0 {
		item.SetAttribute("subscription", ri.Subscription)
	}
	if ri.Ask {
		item.SetAttribute("ask", "subscribe")
	}
	for _, group := range ri.Groups {
		if len(group) == 0 {
			continue
		}
		gr := xml.NewElementName("group")
		gr.SetText(group)
		item.AppendElement(gr)
	}
	return item
}

func (r *ModRoster) parseVer(ver string) int {
	if len(ver) > 0 && ver[0] == 'v' {
		v, _ := strconv.Atoi(ver[1:])
		return v
	}
	return 0
}
