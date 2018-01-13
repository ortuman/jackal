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

type userServerUnit struct {
	contactUnit *contactServerUnit
}

type contactServerUnit struct {
	userUnit *userServerUnit
}

func (u *userServerUnit) updateRosterItem(ri *storage.RosterItem) {
}

func (u *userServerUnit) processPresence(presence *xml.Presence) {
}

type Roster struct {
	queue concurrent.OperationQueue
	strm  stream.C2SStream

	userUnit    *userServerUnit
	contactUnit *contactServerUnit

	requestedMu sync.RWMutex
	requested   bool
}

func NewRoster(strm stream.C2SStream) *Roster {
	r := &Roster{
		queue: concurrent.OperationQueue{
			QueueSize: 32,
			Timeout:   time.Second * 5,
		},
		strm: strm,
	}
	r.userUnit = &userServerUnit{}
	r.contactUnit = &contactServerUnit{}
	r.userUnit.contactUnit = r.contactUnit
	r.contactUnit.userUnit = r.userUnit
	return r
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

func (r *Roster) DeliverPendingApprovalNotifications() {
	r.queue.Async(func() {
		r.deliverPendingApprovalNotifications()
	})
}

func (r *Roster) BrodcastPresence(presence *xml.Presence) {
	r.queue.Async(func() {
		r.brodcastPresence(presence)
	})
}

func (r *Roster) ProcessPresence(presence *xml.Presence) {
	r.queue.Async(func() {
		r.processPresence(presence)
	})
}

func (r *Roster) brodcastPresence(presence *xml.Presence) {
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

func (r *Roster) processPresence(presence *xml.Presence) {
	// var err error
	switch presence.Type() {
	case xml.SubscribeType:
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
	r.userUnit.updateRosterItem(ri)
	r.strm.SendElement(iq.ResultIQ())
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
