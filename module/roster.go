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
	ri, err := r.newRosterItemElement(items[0])
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

func (r *Roster) removeRosterItem(rosterItem *storage.RosterItem) error {
	// https://xmpp.org/rfcs/rfc3921.html#int-remove
	username := r.strm.Username()
	resource := r.strm.Resource()

	contactJID := rosterItem.JID

	log.Infof("removing roster item: %s (%s/%s)", contactJID.ToBareJID(), username, resource)

	if err := storage.Instance().DeleteRosterNotification(username, contactJID.ToBareJID()); err != nil {
		return err
	}
	if err := storage.Instance().DeleteRosterItem(username, contactJID.ToBareJID()); err != nil {
		return err
	}
	r.pushRosterItem(rosterItem, r.strm.JID())
	return nil
}

func (r *Roster) updateRosterItem(rosterItem *storage.RosterItem) error {
	username := r.strm.Username()
	resource := r.strm.Resource()

	jid := rosterItem.JID.ToBareJID()

	log.Infof("inserting/updating roster item: %s (%s/%s)", jid, username, resource)

	ri, err := storage.Instance().FetchRosterItem(username, jid)
	if err != nil {
		return err
	}
	if ri != nil {
		// update roster item
		if len(rosterItem.Name) > 0 {
			ri.Name = rosterItem.Name
		}
		ri.Groups = rosterItem.Groups

	} else {
		ri = &storage.RosterItem{
			Username:     username,
			JID:          rosterItem.JID,
			Name:         rosterItem.Name,
			Subscription: subscriptionNone,
			Groups:       rosterItem.Groups,
			Ask:          rosterItem.Ask,
		}
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(ri); err != nil {
		return err
	}
	r.pushRosterItem(ri, r.strm.JID())
	return nil
}

func (r *Roster) processSubscribe(presence *xml.Presence) error {
	username := r.strm.Username()
	res := r.strm.Resource()

	userJID := r.strm.JID()
	contactJID := presence.ToJID()

	ri, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.ToBareJID())
	if err != nil {
		return err
	}
	if ri != nil {
		switch ri.Subscription {
		case subscriptionTo, subscriptionBoth:
			// already subscribed
			return nil

		default:
			ri.Ask = true
		}
	} else {
		// create roster item if not previously created
		ri = &storage.RosterItem{
			Username:     username,
			JID:          contactJID,
			Subscription: subscriptionNone,
			Ask:          true,
		}
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(ri); err != nil {
		return err
	}
	r.pushRosterItem(ri, r.strm.JID())

	log.Infof("authorization requested: %v -> %v (%s/%s)", userJID.ToBareJID(), contactJID, username, res)

	// send presence approval notification to contact
	p := xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.SubscribeType)
	p.AppendElements(presence.Elements()...)

	// archive roster approval notification
	err = storage.Instance().InsertOrUpdateRosterNotification(username, contactJID.ToBareJID(), p)
	if err != nil {
		return err
	}
	r.routeElement(p, contactJID)
	return nil
}

func (r *Roster) processSubscribed(presence *xml.Presence) error {
	username := r.strm.Username()
	res := r.strm.Resource()

	userJID := presence.ToJID()
	contactJID := r.strm.JID()

	contactRosterItem, err := storage.Instance().FetchRosterItem(contactJID.Node(), userJID.ToBareJID())
	if err != nil {
		return err
	}
	userRosterItem, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.ToBareJID())
	if err != nil {
		return err
	}
	if contactRosterItem == nil || userRosterItem == nil {
		// silently ignore
		return nil
	}
	log.Infof("authorization granted: %v <- %v (%s/%s)", userJID.ToBareJID(), contactJID, username, res)

	// remove approval notification
	if err := storage.Instance().DeleteRosterNotification(userJID.Node(), contactJID.ToBareJID()); err != nil {
		return err
	}

	// update contact's roster item...
	switch contactRosterItem.Subscription {
	case subscriptionTo:
		contactRosterItem.Subscription = subscriptionBoth
	case subscriptionNone:
		contactRosterItem.Subscription = subscriptionFrom
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(contactRosterItem); err != nil {
		return err
	}
	r.pushRosterItem(contactRosterItem, contactJID)

	// update user's roster item...
	if userRosterItem != nil {
		switch userRosterItem.Subscription {
		case subscriptionFrom:
			userRosterItem.Subscription = subscriptionBoth
		case subscriptionNone:
			userRosterItem.Subscription = subscriptionTo
		}
		userRosterItem.Ask = false
		if err := storage.Instance().InsertOrUpdateRosterItem(userRosterItem); err != nil {
			return err
		}
	}

	// send 'subscribed' presence to user...
	p := xml.NewPresence(contactJID.ToBareJID(), userJID.ToBareJID(), xml.SubscribedType)
	p.AppendElements(presence.Elements()...)
	r.routeElement(p, userJID)

	// send available presence from all of the contact's available resources to the user
	contactStreams := stream.C2S().AvailableStreams(contactJID.Node())
	for _, contactStream := range contactStreams {
		p := xml.NewPresence(contactStream.JID().ToFullJID(), userJID.ToBareJID(), xml.AvailableType)
		r.routeElement(p, userJID)
	}
	return nil
}

func (r *Roster) processUnsubscribe(presence *xml.Presence) error {
	username := r.strm.Username()
	res := r.strm.Resource()

	userJID := r.strm.JID()
	contactJID := presence.ToJID()

	log.Infof("authorization cancelled: %v <- %v (%s/%s)", userJID.ToBareJID(), contactJID, username, res)

	userRosterItem, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.ToBareJID())
	if err != nil {
		return err
	}
	contactRosterItem, err := storage.Instance().FetchRosterItem(contactJID.Node(), userJID.ToBareJID())
	if err != nil {
		return err
	}
	if userRosterItem == nil {
		// silently ignore
		return nil
	}
	switch userRosterItem.Subscription {
	case subscriptionBoth:
		userRosterItem.Subscription = subscriptionFrom
	default:
		userRosterItem.Subscription = subscriptionNone
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(userRosterItem); err != nil {
		return err
	}
	r.pushRosterItem(userRosterItem, userJID)

	if contactRosterItem != nil {
		switch contactRosterItem.Subscription {
		case subscriptionBoth:
			contactRosterItem.Subscription = subscriptionTo
		default:
			contactRosterItem.Subscription = subscriptionNone
		}
		if err := storage.Instance().InsertOrUpdateRosterItem(contactRosterItem); err != nil {
			return err
		}
		r.pushRosterItem(contactRosterItem, contactJID)
	}
	// route the presence stanza of type "unsubscribe" to the contact
	p := xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribeType)
	r.routeElement(p, userJID)

	return nil
}

func (r *Roster) processUnsubscribed(presence *xml.Presence) error {
	username := r.strm.Username()
	res := r.strm.Resource()

	userJID := presence.ToJID()
	contactJID := r.strm.JID()

	log.Infof("authorization revoked: %v <- %v (%s/%s)", userJID.ToBareJID(), contactJID, username, res)

	contactRosterItem, err := storage.Instance().FetchRosterItem(contactJID.Node(), userJID.ToBareJID())
	if err != nil {
		return err
	}
	userRosterItem, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.ToBareJID())
	if err != nil {
		return err
	}
	if contactRosterItem != nil {
		switch contactRosterItem.Subscription {
		case subscriptionBoth:
			contactRosterItem.Subscription = subscriptionTo
		default:
			contactRosterItem.Subscription = subscriptionNone
		}
		if err := storage.Instance().InsertOrUpdateRosterItem(contactRosterItem); err != nil {
			return err
		}
		r.pushRosterItem(contactRosterItem, contactJID)
	}
	if userRosterItem != nil {
		switch userRosterItem.Subscription {
		case subscriptionBoth:
			userRosterItem.Subscription = subscriptionFrom
		default:
			userRosterItem.Subscription = subscriptionNone
		}
		if err := storage.Instance().InsertOrUpdateRosterItem(userRosterItem); err != nil {
			return err
		}
		r.pushRosterItem(userRosterItem, userJID)
	}

	// route the presence stanza of type "unsubscribed" to the user
	p := xml.NewPresence(contactJID.ToBareJID(), userJID.ToBareJID(), xml.UnsubscribedType)
	r.routeElement(p, userJID)

	// send 'unavailable' presence from all of the contact's available resources to the user
	contactStreams := stream.C2S().AvailableStreams(contactJID.Node())
	for _, contactStream := range contactStreams {
		p := xml.NewPresence(contactStream.JID().ToFullJID(), userJID.ToBareJID(), xml.UnavailableType)
		r.routeElement(p, userJID)
	}
	return nil
}

func (r *Roster) pushRosterItem(item *storage.RosterItem, to *xml.JID) {
	if stream.C2S().IsLocalDomain(to.Domain()) {
		query := xml.NewElementNamespace("query", rosterNamespace)
		query.AppendElement(r.elementFromRosterItem(item))

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

func (r *Roster) newRosterItemElement(item xml.Element) (*storage.RosterItem, error) {
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
