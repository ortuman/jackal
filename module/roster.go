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
	queue       concurrent.OperationQueue
	strm        stream.C2SStream
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
		r.deliverPendingApprovalNotifications()
	})
}

func (r *Roster) BrodcastPresence(presence *xml.Presence) {
	r.queue.Async(func() {
	})
}

func (r *Roster) processPresence(presence *xml.Presence) {
	var err error
	switch presence.Type() {
	case xml.SubscribeType:
		err = r.processSubscribe(presence)
	case xml.SubscribedType:
		err = r.processSubscribed(presence)
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
	return nil
}

func (r *Roster) updateRosterItem(ri *storage.RosterItem) error {
	userJID := r.strm.JID()
	contactJID := ri.JID

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
			Username:     r.strm.Username(),
			JID:          ri.JID,
			Name:         ri.Name,
			Subscription: subscriptionNone,
			Groups:       ri.Groups,
			Ask:          ri.Ask,
		}
	}
	if err := r.insertOrUpdateRosterItem(userRi); err != nil {
		return err
	}
	r.pushRosterItem(userRi, r.strm.JID())
	return nil
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
			Username:     userJID.Node(),
			JID:          contactJID,
			Subscription: subscriptionNone,
			Ask:          true,
		}
	}
	if err := r.insertOrUpdateRosterItem(ri); err != nil {
		return err
	}
	r.pushRosterItem(ri, userJID)

	// stamp the presence stanza of type "subscribe" with the user's bare JID as the 'from' address
	p := xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.SubscribeType)
	p.AppendElements(presence.Elements())

	if r.isLocalJID(contactJID) {
		// archive roster approval notification
		if err := r.insertOrUpdateRosterNotification(userJID, contactJID, p); err != nil {
			return err
		}
		r.routePresence(p, contactJID)

	} else {
		r.routePresenceRemotely(p, contactJID)
	}
	return nil
}

func (r *Roster) processSubscribed(presence *xml.Presence) error {
	userJID := presence.ToJID()
	contactJID := r.strm.JID()

	log.Infof("processing 'subscribed' - user: %s (%s/%s)", contactJID, r.strm.Username(), r.strm.Resource())

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
		r.pushRosterItem(contactRi, contactJID)
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
			r.pushRosterItem(userRi, userJID)

			r.routePresence(p, userJID)
			r.routePresencesFrom(contactJID, userJID, xml.AvailableType)
		}

	} else {
		r.routePresenceRemotely(p, userJID)
		r.routeRemotelyPresencesFrom(contactJID, userJID, xml.AvailableType)
	}
	return nil
}

func (r *Roster) processUnsubscribed(presence *xml.Presence) error {
	userJID := presence.ToJID()
	contactJID := r.strm.JID()

	log.Infof("processing 'unsubscribed' - user: %s (%s/%s)", contactJID, r.strm.Username(), r.strm.Resource())

	if err := r.deleteRosterNotification(userJID, contactJID); err != nil {
		return err
	}
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
		r.pushRosterItem(contactRi, contactJID)
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
			r.pushRosterItem(userRi, userJID)

			r.routePresence(p, userJID)
			r.routePresencesFrom(contactJID, userJID, xml.UnavailableType)
		}

	} else {
		r.routePresenceRemotely(p, userJID)
		r.routeRemotelyPresencesFrom(contactJID, userJID, xml.UnavailableType)
	}
	return nil
}

func (r *Roster) fetchRosterItem(userJID *xml.JID, contactJID *xml.JID) (*storage.RosterItem, error) {
	ri, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.ToBareJID())
	if err != nil {
		return nil, err
	}
	return ri, nil
}

func (r *Roster) insertOrUpdateRosterNotification(userJID *xml.JID, contactJID *xml.JID, presence *xml.Presence) error {
	return storage.Instance().InsertOrUpdateRosterNotification(userJID.Node(), contactJID.ToBareJID(), presence)
}

func (r *Roster) deleteRosterNotification(userJID *xml.JID, contactJID *xml.JID) error {
	return storage.Instance().DeleteRosterNotification(userJID.Node(), contactJID.ToBareJID())
}

func (r *Roster) insertOrUpdateRosterItem(ri *storage.RosterItem) error {
	if err := storage.Instance().InsertOrUpdateRosterItem(ri); err != nil {
		return err
	}
	return nil
}

func (r *Roster) pushRosterItem(ri *storage.RosterItem, to *xml.JID) {
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

func (r *Roster) isLocalJID(jid *xml.JID) bool {
	return stream.C2S().IsLocalDomain(jid.Domain())
}

func (r *Roster) routePresencesFrom(from *xml.JID, to *xml.JID, presenceType string) {
	strms := stream.C2S().AvailableStreams(from.Node())
	for _, strm := range strms {
		p := xml.NewPresence(strm.JID().ToFullJID(), to.ToBareJID(), presenceType)
		r.routePresence(p, to)
	}
}

func (r *Roster) routeRemotelyPresencesFrom(from *xml.JID, to *xml.JID, presenceType string) {
	fromStreams := stream.C2S().AvailableStreams(from.Node())
	for _, fromStream := range fromStreams {
		p := xml.NewPresence(fromStream.JID().ToFullJID(), to.ToBareJID(), presenceType)
		r.routePresenceRemotely(p, to)
	}
}

func (r *Roster) routePresence(presence *xml.Presence, to *xml.JID) {
	toStreams := stream.C2S().AvailableStreams(to.Node())
	for _, toStream := range toStreams {
		p := xml.NewPresence(presence.From(), toStream.JID().ToFullJID(), presence.Type())
		p.AppendElements(presence.Elements())
		toStream.SendElement(presence)
	}
}

func (r *Roster) routePresenceRemotely(presence *xml.Presence, to *xml.JID) {
	// TODO(ortuman): Implement XMPP federation
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
		if len(group) == 0 {
			continue
		}
		gr := xml.NewElementName("group")
		gr.SetText(group)
		item.AppendElement(gr)
	}
	return item
}
