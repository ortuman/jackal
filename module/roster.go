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
		err = r.userSubscribe(presence)
	case xml.SubscribedType:
		err = r.contactSubscribed(presence)
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
	userJID := r.strm.JID()
	contactJID := ri.JID

	userRi, contactRi, err := r.fetchRosterItem(userJID, contactJID)
	if err != nil {
		return err
	}
	if userRi == nil {
		return nil
	}
	log.Infof("removing roster item: %s (%s/%s)", contactJID.ToBareJID(), r.strm.Username(), r.strm.Resource())

	// route the presence stanza of type "unsubscribe" and "unsubscribed" to the contact
	if userRi.Subscription == subscriptionFrom || userRi.Subscription == subscriptionBoth {
		r.routeElement(xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribeType), contactJID)
		r.routeElement(xml.NewPresence(userJID.ToBareJID(), contactJID.ToBareJID(), xml.UnsubscribedType), contactJID)
	}

	if err := storage.Instance().DeleteRosterNotification(userJID.Node(), contactJID.ToBareJID()); err != nil {
		return err
	}
	if err := storage.Instance().DeleteRosterItem(userJID.Node(), contactJID.ToBareJID()); err != nil {
		return err
	}
	r.pushRosterItem(ri, userJID)

	// send unavailable presence from all of the users's available resources to the contact
	if userRi.Subscription == subscriptionFrom || userRi.Subscription == subscriptionBoth {
		r.sendPresencesFrom(userJID, contactJID, xml.UnavailableType)
	}

	if contactRi != nil {
		switch contactRi.Subscription {
		case subscriptionBoth:
			contactRi.Subscription = subscriptionTo
			r.pushRosterItem(contactRi, contactJID)
			fallthrough

		default:
			contactRi.Subscription = subscriptionNone
			if err := storage.Instance().InsertOrUpdateRosterItem(contactRi); err != nil {
				return err
			}
			r.pushRosterItem(contactRi, contactJID)
		}
		// send unavailable presence from all of the contact's available resources to the user
		if contactRi.Subscription == subscriptionFrom || contactRi.Subscription == subscriptionBoth {
			r.sendPresencesFrom(contactJID, userJID, xml.UnavailableType)
		}
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

func (r *Roster) userSubscribe(presence *xml.Presence) error {
	userJID := r.strm.JID()
	contactJID := presence.ToJID()

	log.Infof("processing 'subscribe' - contact: %s (%s/%s)", contactJID, r.strm.Username(), r.strm.Resource())

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

func (r *Roster) contactSubscribed(presence *xml.Presence) error {
	return nil
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
