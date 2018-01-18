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
	queue     concurrent.OperationQueue
	strm      stream.C2SStream
	lock      sync.RWMutex
	requested bool
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
	r.lock.RLock()
	defer r.lock.RUnlock()
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
	})
}

func (r *Roster) DeliverPendingApprovalNotifications() {
	r.queue.Async(func() {
		if err := r.deliverPendingApprovalNotifications(); err != nil {
			log.Error(err)
		}
	})
}

func (r *Roster) ReceivePresences() {
	r.queue.Async(func() {
		if err := r.receiveRosterPresences(); err != nil {
			log.Error(err)
		}
	})
}

func (r *Roster) BroadcastPresence(presence *xml.Presence) {
	r.queue.Async(func() {
		if err := r.broadcastPresence(presence); err != nil {
			log.Error(err)
		}
	})
}

func (r *Roster) deliverPendingApprovalNotifications() error {
	rosterNotifications, err := storage.Instance().FetchRosterNotifications(r.strm.Username())
	if err != nil {
		return err
	}

	for _, rosterNotification := range rosterNotifications {
		fromJID, err := xml.NewJID(rosterNotification.User, r.strm.Domain(), "", true)
		if err != nil {
			return err
		}
		p := xml.NewPresence(fromJID, r.strm.JID(), xml.SubscribeType)
		p.AppendElements(rosterNotification.Elements)
		r.strm.SendElement(p)
	}
	return nil
}

func (r *Roster) receiveRosterPresences() error {
	items, err := storage.Instance().FetchRosterItemsAsUser(r.strm.Username())
	if err != nil {
		return err
	}
	userJID := r.strm.JID()
	for _, item := range items {
		switch item.Subscription {
		case subscriptionTo, subscriptionBoth:
			itemJID, err := r.rosterItemJID(&item)
			if err != nil {
				return err
			}
			r.routePresencesFrom(itemJID, userJID, xml.AvailableType)
		default:
			break
		}
	}
	return nil
}

func (r *Roster) broadcastPresence(presence *xml.Presence) error {
	contactJID := presence.FromJID()
	items, err := storage.Instance().FetchRosterItemsAsContact(contactJID.Node())
	if err != nil {
		return err
	}
	for _, item := range items {
		switch item.Subscription {
		case subscriptionTo, subscriptionBoth:
			break
		default:
			continue
		}
		jidStr := fmt.Sprintf("%s@%s", item.User, contactJID.Domain())
		userJID, err := xml.NewJIDString(jidStr, true)
		if err != nil {
			return err
		}
		r.routePresence(presence, userJID)
	}
	return nil
}

func (r *Roster) sendRoster(iq *xml.IQ, query xml.Element) {
	if query.ElementsCount() > 0 {
		r.strm.SendElement(iq.BadRequestError())
		return
	}
	log.Infof("retrieving user roster... (%s/%s)", r.strm.Username(), r.strm.Resource())

	result := iq.ResultIQ()
	q := xml.NewElementNamespace("query", rosterNamespace)

	items, err := storage.Instance().FetchRosterItemsAsUser(r.strm.Username())
	if err != nil {
		log.Error(err)
		r.strm.SendElement(iq.InternalServerError())
		return
	}
	if items != nil {
		for _, item := range items {
			elem, err := r.elementFromRosterItem(&item)
			if err != nil {
				log.Error(err)
				r.strm.SendElement(iq.BadRequestError())
				return
			}
			q.AppendElement(elem)
		}
	}
	result.AppendElement(q)
	r.strm.SendElement(result)

	r.lock.Lock()
	r.requested = true
	r.lock.Unlock()
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
		if err := r.removeRosterItem(ri); err != nil {
			log.Error(err)
			r.strm.SendElement(iq.InternalServerError())
			return
		}
	default:
		if err := r.updateRosterItem(ri); err != nil {
			log.Error(err)
			r.strm.SendElement(iq.InternalServerError())
			return
		}
	}
	r.strm.SendElement(iq.ResultIQ())
}

func (r *Roster) removeRosterItem(ri *storage.RosterItem) error {
	userJID := r.strm.JID()
	contactJID, err := r.rosterItemJID(ri)
	if err != nil {
		return err
	}

	log.Infof("removing roster item: %v (%s/%s)", contactJID, r.strm.Username(), r.strm.Resource())

	var unsubscribe *xml.Presence
	var unsubscribed *xml.Presence

	userRi, err := r.fetchRosterItem(userJID, contactJID)
	if err != nil {
		return err
	}
	userSubscription := subscriptionNone
	if userRi != nil {
		userSubscription = userRi.Subscription
		switch userSubscription {
		case subscriptionTo:
			unsubscribe = xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribeType)
		case subscriptionFrom:
			unsubscribed = xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribedType)
		case subscriptionBoth:
			unsubscribe = xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribeType)
			unsubscribed = xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribedType)
		}
		userRi.Subscription = subscriptionRemove
		userRi.Ask = false

		if err := r.deleteRosterNotification(userJID, contactJID); err != nil {
			return err
		}
		if err := r.deleteRosterItem(userJID, contactJID); err != nil {
			return err
		}
		if err := r.pushRosterItem(userRi, userJID); err != nil {
			return err
		}
	}

	if r.isLocalJID(contactJID) {
		contactRi, err := r.fetchRosterItem(contactJID, userJID)
		if err != nil {
			return err
		}
		if contactRi != nil {
			if contactRi.Subscription == subscriptionFrom || contactRi.Subscription == subscriptionBoth {
				r.routePresencesFrom(contactJID, userJID, xml.UnavailableType)
			}
			switch contactRi.Subscription {
			case subscriptionBoth:
				contactRi.Subscription = subscriptionTo
				if err := r.pushRosterItem(contactRi, contactJID); err != nil {
					return err
				}
				fallthrough
			default:
				contactRi.Subscription = subscriptionNone
				if err := r.pushRosterItem(contactRi, contactJID); err != nil {
					return err
				}
			}
			if err := r.insertOrUpdateRosterItem(contactRi); err != nil {
				return err
			}
		}
	}
	if unsubscribe != nil {
		r.routePresence(unsubscribe, contactJID)
	}
	if unsubscribed != nil {
		r.routePresence(unsubscribed, contactJID)
	}
	if userSubscription == subscriptionFrom || userSubscription == subscriptionBoth {
		r.routePresencesFrom(userJID, contactJID, xml.UnavailableType)
	}
	return nil
}

func (r *Roster) updateRosterItem(ri *storage.RosterItem) error {
	userJID := r.strm.JID()
	contactJID, err := r.rosterItemJID(ri)
	if err != nil {
		return err
	}

	log.Infof("updating roster item - contact: %s (%s/%s)", contactJID, r.strm.Username(), r.strm.Resource())

	userRi, err := r.fetchRosterItem(userJID, contactJID)
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
			User:         r.strm.Username(),
			Contact:      ri.Contact,
			Name:         ri.Name,
			Subscription: subscriptionNone,
			Groups:       ri.Groups,
			Ask:          ri.Ask,
		}
	}
	if err := r.insertOrUpdateRosterItem(userRi); err != nil {
		return err
	}
	return r.pushRosterItem(userRi, r.strm.JID())
}

func (r *Roster) processSubscribe(presence *xml.Presence) error {
	userJID := r.strm.JID()
	contactJID := presence.ToJID()

	log.Infof("processing 'subscribe' - contact: %s (%s/%s)", contactJID, r.strm.Username(), r.strm.Resource())

	ri, err := r.fetchRosterItem(userJID, contactJID)
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
			User:         userJID.Node(),
			Contact:      contactJID.Node(),
			Subscription: subscriptionNone,
			Ask:          true,
		}
	}
	if err := r.insertOrUpdateRosterItem(ri); err != nil {
		return err
	}
	if err := r.pushRosterItem(ri, userJID); err != nil {
		return err
	}

	// stamp the presence stanza of type "subscribe" with the user's bare JID as the 'from' address
	p := xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.SubscribeType)
	p.AppendElements(presence.Elements())

	if r.isLocalJID(contactJID) {
		// archive roster approval notification
		if err := r.insertOrUpdateRosterNotification(userJID, contactJID, p); err != nil {
			return err
		}
	}
	r.routePresence(p, contactJID)
	return nil
}

func (r *Roster) processSubscribed(presence *xml.Presence) error {
	userJID := presence.ToJID()
	contactJID := r.strm.JID()

	log.Infof("processing 'subscribed' - user: %s (%s/%s)", userJID, r.strm.Username(), r.strm.Resource())

	if err := r.deleteRosterNotification(userJID, contactJID); err != nil {
		return err
	}
	contactRi, err := r.fetchRosterItem(contactJID, userJID)
	if err != nil {
		return err
	}
	if contactRi != nil {
		switch contactRi.Subscription {
		case subscriptionTo:
			contactRi.Subscription = subscriptionBoth
		case subscriptionNone:
			contactRi.Subscription = subscriptionFrom
		}
		if err := r.insertOrUpdateRosterItem(contactRi); err != nil {
			return err
		}
		if err := r.pushRosterItem(contactRi, contactJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "subscribed" with the contact's bare JID as the 'from' address
	p := xml.NewPresence(contactJID.ToBareJID(), userJID.ToBareJID(), xml.SubscribedType)
	p.AppendElements(presence.Elements())

	if r.isLocalJID(userJID) {
		userRi, err := r.fetchRosterItem(userJID, contactJID)
		if err != nil {
			return err
		}
		if userRi != nil {
			switch userRi.Subscription {
			case subscriptionFrom:
				userRi.Subscription = subscriptionBoth
			case subscriptionNone:
				userRi.Subscription = subscriptionTo
			default:
				return nil
			}
			userRi.Ask = false
			if err := r.insertOrUpdateRosterItem(userRi); err != nil {
				return err
			}
			if err := r.pushRosterItem(userRi, userJID); err != nil {
				return err
			}
		}
	}
	r.routePresence(p, userJID)
	r.routePresencesFrom(contactJID, userJID, xml.AvailableType)
	return nil
}

func (r *Roster) processUnsubscribe(presence *xml.Presence) error {
	userJID := r.strm.JID()
	contactJID := presence.ToJID()

	log.Infof("processing 'unsubscribe' - contact: %s (%s/%s)", contactJID, r.strm.Username(), r.strm.Resource())

	userRi, err := r.fetchRosterItem(userJID, contactJID)
	if err != nil {
		return err
	}
	userSubscription := subscriptionNone
	if userRi != nil {
		userSubscription = userRi.Subscription
		switch userSubscription {
		case subscriptionBoth:
			userRi.Subscription = subscriptionFrom
		default:
			userRi.Subscription = subscriptionNone
		}
		if err := r.insertOrUpdateRosterItem(userRi); err != nil {
			return err
		}
		if err := r.pushRosterItem(userRi, userJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "unsubscribe" with the users's bare JID as the 'from' address
	p := xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribeType)
	p.AppendElements(presence.Elements())

	if r.isLocalJID(contactJID) {
		contactRi, err := r.fetchRosterItem(contactJID, userJID)
		if err != nil {
			return err
		}
		if contactRi != nil {
			switch contactRi.Subscription {
			case subscriptionBoth:
				contactRi.Subscription = subscriptionTo
			default:
				contactRi.Subscription = subscriptionNone
			}
			if err := r.insertOrUpdateRosterItem(contactRi); err != nil {
				return err
			}
			if err := r.pushRosterItem(contactRi, contactJID); err != nil {
				return err
			}
		}
	}
	r.routePresence(p, contactJID)

	if userSubscription == subscriptionTo || userSubscription == subscriptionBoth {
		r.routePresencesFrom(contactJID, userJID, xml.UnavailableType)
	}
	return nil
}

func (r *Roster) processUnsubscribed(presence *xml.Presence) error {
	userJID := presence.ToJID()
	contactJID := r.strm.JID()

	log.Infof("processing 'unsubscribed' - user: %s (%s/%s)", userJID, r.strm.Username(), r.strm.Resource())

	if err := r.deleteRosterNotification(userJID, contactJID); err != nil {
		return err
	}
	contactRi, err := r.fetchRosterItem(contactJID, userJID)
	if err != nil {
		return err
	}
	contactSubscription := subscriptionNone
	if contactRi != nil {
		contactSubscription = contactRi.Subscription
		switch contactSubscription {
		case subscriptionBoth:
			contactRi.Subscription = subscriptionTo
		default:
			contactRi.Subscription = subscriptionNone
		}
		if err := r.insertOrUpdateRosterItem(contactRi); err != nil {
			return err
		}
		if err := r.pushRosterItem(contactRi, contactJID); err != nil {
			return err
		}
	}
	// stamp the presence stanza of type "unsubscribed" with the contact's bare JID as the 'from' address
	p := xml.NewPresence(contactJID.ToBareJID(), userJID.ToBareJID(), xml.UnsubscribedType)
	p.AppendElements(presence.Elements())

	if r.isLocalJID(userJID) {
		userRi, err := r.fetchRosterItem(userJID, contactJID)
		if err != nil {
			return err
		}
		if userRi != nil {
			switch userRi.Subscription {
			case subscriptionBoth:
				userRi.Subscription = subscriptionFrom
			default:
				userRi.Subscription = subscriptionNone
			}
			userRi.Ask = false
			if err := r.insertOrUpdateRosterItem(userRi); err != nil {
				return err
			}
			if err := r.pushRosterItem(userRi, userJID); err != nil {
				return err
			}
		}
	}
	r.routePresence(p, userJID)

	if contactSubscription == subscriptionFrom || contactSubscription == subscriptionBoth {
		r.routePresencesFrom(contactJID, userJID, xml.UnavailableType)
	}
	return nil
}

func (r *Roster) insertOrUpdateRosterNotification(userJID *xml.JID, contactJID *xml.JID, presence *xml.Presence) error {
	rn := &storage.RosterNotification{
		User:     userJID.Node(),
		Contact:  contactJID.Node(),
		Elements: presence.Elements(),
	}
	return storage.Instance().InsertOrUpdateRosterNotification(rn)
}

func (r *Roster) deleteRosterNotification(userJID *xml.JID, contactJID *xml.JID) error {
	return storage.Instance().DeleteRosterNotification(userJID.Node(), contactJID.Node())
}

func (r *Roster) fetchRosterItem(userJID *xml.JID, contactJID *xml.JID) (*storage.RosterItem, error) {
	ri, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.Node())
	if err != nil {
		return nil, err
	}
	return ri, nil
}

func (r *Roster) insertOrUpdateRosterItem(ri *storage.RosterItem) error {
	if err := storage.Instance().InsertOrUpdateRosterItem(ri); err != nil {
		return err
	}
	return nil
}

func (r *Roster) deleteRosterItem(userJID *xml.JID, contactJID *xml.JID) error {
	return storage.Instance().DeleteRosterItem(userJID.Node(), contactJID.Node())
}

func (r *Roster) pushRosterItem(ri *storage.RosterItem, to *xml.JID) error {
	elem, err := r.elementFromRosterItem(ri)
	if err != nil {
		return err
	}
	query := xml.NewElementNamespace("query", rosterNamespace)
	query.AppendElement(elem)

	streams := stream.C2S().AvailableStreams(to.Node())
	for _, strm := range streams {
		if !strm.IsRosterRequested() {
			continue
		}
		pushEl := xml.NewIQType(uuid.New(), xml.SetType)
		pushEl.SetTo(strm.JID().String())
		pushEl.AppendElement(query)
		strm.SendElement(pushEl)
	}
	return nil
}

func (r *Roster) isLocalJID(jid *xml.JID) bool {
	return stream.C2S().IsLocalDomain(jid.Domain())
}

func (r *Roster) routePresencesFrom(from *xml.JID, to *xml.JID, presenceType string) {
	fromStreams := stream.C2S().AvailableStreams(from.Node())
	for _, fromStream := range fromStreams {
		p := xml.NewPresence(fromStream.JID(), to.ToBareJID(), presenceType)
		if presenceType == xml.AvailableType {
			p.AppendElements(fromStream.PresenceElements())
		}
		r.routePresence(p, to)
	}
}

func (r *Roster) routePresence(presence *xml.Presence, to *xml.JID) {
	if stream.C2S().IsLocalDomain(to.Domain()) {
		toStreams := stream.C2S().AvailableStreams(to.Node())
		for _, toStream := range toStreams {
			p := xml.NewPresence(presence.FromJID(), toStream.JID(), presence.Type())
			p.AppendElements(presence.Elements())
			toStream.SendElement(p)
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
		ri.Contact = j.Node()
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

func (r *Roster) elementFromRosterItem(ri *storage.RosterItem) (xml.Element, error) {
	riJID, err := r.rosterItemJID(ri)
	if err != nil {
		return nil, err
	}
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
	return item, nil
}

func (r *Roster) rosterItemJID(ri *storage.RosterItem) (*xml.JID, error) {
	return xml.NewJIDString(fmt.Sprintf("%s@%s", ri.Contact, r.strm.Domain()), true)
}
