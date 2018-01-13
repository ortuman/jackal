/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ortuman/jackal/concurrent"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const rosterNamespace = "jabber:iq:roster"

const (
	subscriptionNone   = "none"
	subscriptionFrom   = "from"
	subscriptionTo     = "to"
	subscriptionBoth   = "both"
	subscriptionRemove = "remove"
)

type Roster struct {
	queue concurrent.OperationQueue
	strm  stream.C2SStream
	once  sync.Once

	requestedMu sync.RWMutex
	requested   bool
}

func NewRoster(strm stream.C2SStream) *Roster {
	return &Roster{
		queue: concurrent.OperationQueue{
			QueueSize: 32,
			Timeout:   time.Second * 5,
		},
		strm: strm,
	}
}

func (r *Roster) IsRosterRequested() bool {
	r.requestedMu.RLock()
	defer r.requestedMu.RUnlock()
	return r.requested
}

func (r *Roster) AssociatedNamespaces() []string {
	return []string{}
}

func (r *Roster) MatchesIQ(iq *xml.IQ) bool {
	return iq.FindElementNamespace("query", rosterNamespace) != nil
}

func (r *Roster) ProcessIQ(iq *xml.IQ) {
	r.queue.Async(func() {
		q := iq.FindElementNamespace("query", rosterNamespace)
		if iq.IsGet() {
			r.sendRoster(iq, q)
		} else if iq.IsSet() {
			r.updateRoster(iq, q)
		} else {
			r.strm.SendElement(iq.BadRequestError())
		}
	})
}

func (r *Roster) ProcessPresence(presence *xml.Presence) {
	r.queue.Async(func() {
		r.processPresence(presence)
	})
}

func (r *Roster) DeliverPendingApprovalNotifications() {
	r.queue.Async(func() {
		r.once.Do(func() { r.deliverPendingApprovalNotifications() })
	})
}

func (r *Roster) processPresence(presence *xml.Presence) {
	var err error
	switch presence.Type() {
	case xml.SubscribeType:
		err = r.processSubscribe(presence)
	case xml.SubscribedType:
		err = r.processSubscribed(presence)
	case xml.UnsubscribeType:
		err = r.processUnsubscribe(presence)
	case xml.UnsubscribedType:
		err = r.processUnsubscribed(presence)
	}
	if err != nil {
		log.Error(err)
	}
}

func (r *Roster) deliverPendingApprovalNotifications() {
	notifications, err := storage.Instance().FetchRosterNotifications(r.strm.JID().ToBareJID())
	if err != nil {
		log.Error(err)
		return
	}
	for _, notification := range notifications {
		r.strm.SendElement(notification)
	}
}

func (r *Roster) sendRoster(iq *xml.IQ, query xml.Element) {
	if query.ElementsCount() > 0 {
		r.strm.SendElement(iq.BadRequestError())
		return
	}
	log.Infof("retrieving user roster... (%s/%s)", r.strm.Username(), r.strm.Resource())

	result := iq.ResultIQ()
	q := xml.NewElementNamespace("query", rosterNamespace)

	items, err := storage.Instance().FetchRosterItems(r.strm.Username())
	if err != nil {
		log.Error(err)
		r.strm.SendElement(iq.InternalServerError())
		return
	}
	if items != nil {
		for _, item := range items {
			q.AppendElement(r.elementFromRosterItem(&item))
		}
	}
	result.AppendElement(q)
	r.strm.SendElement(result)

	r.requestedMu.Lock()
	r.requested = true
	r.requestedMu.Unlock()
}

func (r *Roster) updateRoster(iq *xml.IQ, query xml.Element) {
	items := query.FindElements("item")
	if len(items) != 1 {
		r.strm.SendElement(iq.BadRequestError())
		return
	}
	ri, err := r.rosterItemFromElement(items[0])
	if err != nil {
		r.strm.SendElement(iq.BadRequestError())
		return
	}
	switch ri.Subscription {
	case subscriptionRemove:
		r.removeRosterItem(ri)
	default:
		r.updateRosterItem(ri)
	}
	r.strm.SendElement(iq.ResultIQ())
}

func (r *Roster) removeRosterItem(ri *storage.RosterItem) error {
	// https://xmpp.org/rfcs/rfc3921.html#int-remove
	userJID := r.strm.JID()
	contactJID := ri.JID

	userRi, contactRi, err := r.fetchRosterItems(userJID, contactJID)
	if err != nil {
		return err
	}
	if userRi == nil {
		return nil
	}
	log.Infof("removing roster item: %s (%s/%s)", contactJID.ToBareJID(), r.strm.Username(), r.strm.Resource())

	// route the presence stanza of type "unsubscribe" and "unsubscribed" to the contact
	r.routeElement(xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribeType), contactJID)
	r.routeElement(xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribedType), contactJID)

	if err := storage.Instance().DeleteRosterNotification(userJID.Node(), contactJID.ToBareJID()); err != nil {
		return err
	}
	if err := storage.Instance().DeleteRosterItem(userJID.Node(), contactJID.ToBareJID()); err != nil {
		return err
	}
	r.pushRosterItem(ri, userJID)

	// send unavailable presence from all of the users's available resources to the conctact
	r.sendAvailablePresencesFrom(userJID, contactJID, xml.UnavailableType)

	if contactRi != nil {
	}
	return nil
}

func (r *Roster) updateRosterItem(ri *storage.RosterItem) error {
	jid := ri.JID.ToBareJID()

	log.Infof("inserting/updating roster item: %s (%s/%s)", jid, r.strm.Username(), r.strm.Resource())

	userRi, err := storage.Instance().FetchRosterItem(r.strm.Username(), jid)
	if err != nil {
		return err
	}
	if userRi != nil {
		// update roster item
		if len(ri.Name) > 0 {
			ri.Name = ri.Name
		}
		ri.Groups = ri.Groups

	} else {
		userRi = &storage.RosterItem{
			Username:     r.strm.Username(),
			JID:          ri.JID,
			Name:         ri.Name,
			Subscription: subscriptionNone,
			Groups:       ri.Groups,
			Ask:          ri.Ask,
		}
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(userRi); err != nil {
		return err
	}
	r.pushRosterItem(userRi, r.strm.JID())
	return nil
}

func (r *Roster) processSubscribe(presence *xml.Presence) error {
	userJID := r.strm.JID()
	contactJID := presence.ToJID()

	log.Infof("processing 'subscribe' - jid: %s (%s/%s)", contactJID, r.strm.Username(), r.strm.Resource())

	ri, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.ToBareJID())
	if err != nil {
		return err
	}
	if ri != nil {
		switch ri.Subscription {
		case subscriptionTo, subscriptionBoth:
			return nil // already subscribed
		default:
			ri.Ask = true
		}
	} else {
		// create roster item if not previously created
		ri = &storage.RosterItem{
			Username:     userJID.Node(),
			JID:          contactJID,
			Subscription: subscriptionNone,
			Ask:          true,
		}
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(ri); err != nil {
		return err
	}
	r.pushRosterItem(ri, userJID)

	// route the presence stanza of type "subscribe" to the contact
	p := xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.SubscribeType)
	p.AppendElements(presence.Elements())
	r.routeElement(p, contactJID)

	// archive roster approval notification
	if stream.C2S().IsLocalDomain(contactJID.Domain()) {
		err = storage.Instance().InsertOrUpdateRosterNotification(userJID.Node(), contactJID.ToBareJID(), p)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Roster) processSubscribed(presence *xml.Presence) error {
	userJID := presence.ToJID()
	contactJID := r.strm.JID()

	userRi, contactRi, err := r.fetchRosterItems(userJID, contactJID)
	if err != nil {
		return err
	}
	log.Infof("processing 'subscribed' - jid: %s (%s/%s)", userJID, r.strm.Username(), r.strm.Resource())

	// remove approval notification
	if err := storage.Instance().DeleteRosterNotification(userJID.Node(), contactJID.ToBareJID()); err != nil {
		return err
	}
	if contactRi != nil {
		switch contactRi.Subscription {
		case subscriptionTo:
			contactRi.Subscription = subscriptionBoth
		case subscriptionNone:
			contactRi.Subscription = subscriptionFrom
		}
		if err := storage.Instance().InsertOrUpdateRosterItem(contactRi); err != nil {
			return err
		}
		r.pushRosterItem(contactRi, contactJID)
	}
	if userRi != nil {
		switch userRi.Subscription {
		case subscriptionFrom:
			userRi.Subscription = subscriptionBoth
		case subscriptionNone:
			userRi.Subscription = subscriptionTo
		}
		userRi.Ask = false
		if err := storage.Instance().InsertOrUpdateRosterItem(userRi); err != nil {
			return err
		}
	}
	// route the presence stanza of type "subscribed" to the contact
	p := xml.NewPresence(contactJID.ToBareJID(), userJID.ToBareJID(), xml.SubscribedType)
	p.AppendElements(presence.Elements())
	r.routeElement(p, userJID)

	// send available presence from all of the contact's available resources to the user
	r.sendAvailablePresencesFrom(contactJID, userJID, xml.AvailableType)
	return nil
}

func (r *Roster) processUnsubscribe(presence *xml.Presence) error {
	userJID := r.strm.JID()
	contactJID := presence.ToJID()

	userRi, contactRi, err := r.fetchRosterItems(userJID, contactJID)
	if err != nil {
		return err
	}
	log.Infof("processing 'unsubscribe' - jid: %s (%s/%s)", contactJID, r.strm.Username(), r.strm.Resource())

	if userRi == nil {
		switch userRi.Subscription {
		case subscriptionBoth:
			userRi.Subscription = subscriptionFrom
		default:
			userRi.Subscription = subscriptionNone
		}
		if err := storage.Instance().InsertOrUpdateRosterItem(userRi); err != nil {
			return err
		}
		r.pushRosterItem(userRi, userJID)
	}
	if contactRi != nil {
		switch contactRi.Subscription {
		case subscriptionBoth:
			contactRi.Subscription = subscriptionTo
		default:
			contactRi.Subscription = subscriptionNone
		}
		if err := storage.Instance().InsertOrUpdateRosterItem(contactRi); err != nil {
			return err
		}
		r.pushRosterItem(contactRi, contactJID)
	}
	// route the presence stanza of type "unsubscribe" to the contact
	p := xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribeType)
	p.AppendElements(presence.Elements())
	r.routeElement(p, contactJID)

	// send 'unavailable' presence from all of the contact's available resources to the user
	r.sendAvailablePresencesFrom(contactJID, userJID, xml.UnavailableType)
	return nil
}

func (r *Roster) processUnsubscribed(presence *xml.Presence) error {
	userJID := presence.ToJID()
	contactJID := r.strm.JID()

	userRi, contactRi, err := r.fetchRosterItems(userJID, contactJID)
	if err != nil {
		return err
	}
	log.Infof("processing 'unsubscribed' presence: %s (%s/%s)", userJID, r.strm.Username(), r.strm.Resource())

	if contactRi != nil {
		switch contactRi.Subscription {
		case subscriptionBoth:
			contactRi.Subscription = subscriptionTo
		default:
			contactRi.Subscription = subscriptionNone
		}
		if err := storage.Instance().InsertOrUpdateRosterItem(contactRi); err != nil {
			return err
		}
		r.pushRosterItem(contactRi, contactJID)
	}
	if userRi != nil {
		switch userRi.Subscription {
		case subscriptionBoth:
			userRi.Subscription = subscriptionFrom
		default:
			userRi.Subscription = subscriptionNone
		}
		if err := storage.Instance().InsertOrUpdateRosterItem(userRi); err != nil {
			return err
		}
		r.pushRosterItem(userRi, userJID)
	}
	// route the presence stanza of type "unsubscribed" to the user
	p := xml.NewPresence(contactJID.ToBareJID(), userJID.ToBareJID(), xml.UnsubscribedType)
	p.AppendElements(presence.Elements())
	r.routeElement(p, userJID)

	// send unavailable presence from all of the contact's available resources to the user
	r.sendAvailablePresencesFrom(contactJID, userJID, xml.UnavailableType)
	return nil
}

func (r *Roster) fetchRosterItems(userJID *xml.JID, contactJID *xml.JID) (*storage.RosterItem, *storage.RosterItem, error) {
	var userRi, contactRi *storage.RosterItem
	var err error

	if stream.C2S().IsLocalDomain(userJID.Domain()) {
		userRi, err = storage.Instance().FetchRosterItem(userJID.Node(), contactJID.ToBareJID())
		if err != nil {
			return nil, nil, err
		}
	}
	if stream.C2S().IsLocalDomain(contactJID.Domain()) {
		contactRi, err = storage.Instance().FetchRosterItem(contactJID.Node(), userJID.ToBareJID())
		if err != nil {
			return nil, nil, err
		}
	}
	return userRi, contactRi, err
}

func (r *Roster) sendAvailablePresencesFrom(from *xml.JID, to *xml.JID, presenceType string) {
	if stream.C2S().IsLocalDomain(from.Domain()) {
		fromStreams := stream.C2S().AvailableStreams(from.Node())
		for _, fromStream := range fromStreams {
			p := xml.NewPresence(fromStream.JID().ToFullJID(), to.ToBareJID(), presenceType)
			r.routeElement(p, to)
		}
	}
}

func (r *Roster) pushRosterItem(ri *storage.RosterItem, to *xml.JID) {
	if stream.C2S().IsLocalDomain(to.Domain()) {
		query := xml.NewElementNamespace("query", rosterNamespace)
		query.AppendElement(r.elementFromRosterItem(ri))

		streams := stream.C2S().AvailableStreams(to.Node())
		for _, strm := range streams {
			if !strm.IsRosterRequested() {
				continue
			}
			pushEl := xml.NewIQType(uuid.New(), xml.SetType)
			pushEl.SetTo(strm.JID().ToFullJID())
			pushEl.AppendElement(query)
			strm.SendElement(pushEl)
		}
	}
}

func (r *Roster) routeElement(element xml.Element, to *xml.JID) {
	if stream.C2S().IsLocalDomain(to.Domain()) {
		streams := stream.C2S().AvailableStreams(to.Node())
		for _, strm := range streams {
			strm.SendElement(element)
		}
	} else {
		// TODO(ortuman): Implement XMPP federation
	}
}

func (r *Roster) rosterItemFromElement(item xml.Element) (*storage.RosterItem, error) {
	ri := &storage.RosterItem{}
	if jid := item.Attribute("jid"); len(jid) > 0 {
		j, err := xml.NewJIDString(jid, false)
		if err != nil {
			return nil, err
		}
		ri.JID = j
	} else {
		return nil, errors.New("item 'jid' attribute is required")
	}
	ri.Name = item.Attribute("name")

	subscription := item.Attribute("subscription")
	if len(subscription) > 0 {
		switch subscription {
		case subscriptionBoth, subscriptionFrom, subscriptionTo, subscriptionNone, subscriptionRemove:
			break
		default:
			return nil, fmt.Errorf("unrecognized 'subscription' enum type: %s", subscription)
		}
		ri.Subscription = subscription
	}
	ask := item.Attribute("ask")
	if len(ask) > 0 {
		if ask != "subscribe" {
			return nil, fmt.Errorf("unrecognized 'ask' enum type: %s", subscription)
		}
		ri.Ask = true
	}
	groups := item.FindElements("group")
	for _, group := range groups {
		if group.AttributesCount() > 0 {
			return nil, errors.New("group element must not contain any attribute")
		}
		ri.Groups = append(ri.Groups, group.Text())
	}
	return ri, nil
}

func (r *Roster) elementFromRosterItem(ri *storage.RosterItem) xml.Element {
	item := xml.NewElementName("item")
	item.SetAttribute("jid", ri.JID.ToBareJID())
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
		gr := xml.NewElementName("group")
		gr.SetText(group)
		item.AppendElement(gr)
	}
	return item
}
