/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"sync"
	"time"

	"github.com/ortuman/jackal/concurrent"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/entity"
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
	strm        Stream
	strmManager StreamManager

	requestedRosterMu sync.RWMutex
	requestedRoster   bool
}

func NewRoster(stream Stream, streamManager StreamManager) *Roster {
	return &Roster{
		queue: concurrent.OperationQueue{
			QueueSize: 32,
			Timeout:   time.Second * 10,
		},
		strm:        stream,
		strmManager: streamManager,
	}
}

func (r *Roster) RequestedRoster() bool {
	r.requestedRosterMu.RLock()
	defer r.requestedRosterMu.RUnlock()
	return r.requestedRoster
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

func (r *Roster) processPresence(presence *xml.Presence) {
	if presence.IsSubscribe() {
		if err := r.subscribeTo(presence.ToJID()); err != nil {
			log.Error(err)
			return
		}
	}
}

func (r *Roster) sendRoster(iq *xml.IQ, query *xml.Element) {
	if query.ElementsCount() > 0 {
		r.strm.SendElement(iq.BadRequestError())
		return
	}
	log.Infof("retrieving user roster... (%s/%s)", r.strm.Username(), r.strm.Resource())

	result := iq.ResultIQ()
	q := xml.NewMutableElementNamespace("query", rosterNamespace)

	items, err := storage.Instance().FetchRosterItems(r.strm.Username())
	if err != nil {
		log.Error(err)
		r.strm.SendElement(iq.InternalServerError())
		return
	}
	if items != nil {
		for _, item := range items {
			q.AppendElement(item.Element())
		}
	}
	result.AppendMutableElement(q)
	r.strm.SendElement(result)

	r.requestedRosterMu.Lock()
	r.requestedRoster = true
	r.requestedRosterMu.Unlock()
}

func (r *Roster) updateRoster(iq *xml.IQ, query *xml.Element) {
	items := query.FindElements("item")
	if len(items) != 1 {
		r.strm.SendElement(iq.BadRequestError())
		return
	}
	ri, err := entity.NewRosterItemElement(items[0])
	if err != nil {
		r.strm.SendElement(iq.BadRequestError())
		return
	}
	updatedRosterItem, err := r.updateRosterItem(ri)
	if err != nil {
		log.Error(err)
		r.strm.SendElement(iq.InternalServerError())
		return
	}
	r.pushRosterItem(updatedRosterItem)
	r.strm.SendElement(iq.ResultIQ())
}

func (r *Roster) subscribeTo(to *xml.JID) error {
	username := r.strm.Username()
	resource := r.strm.Resource()

	jid := to.ToBareJID()

	log.Infof("authorization requested: %s (%s/%s)", jid, username, resource)

	ri, err := storage.Instance().FetchRosterItem(username, jid)
	if err != nil {
		return err
	}
	if ri == nil {
		// create roster item if not previously created
		ri = &entity.RosterItem{
			JID:          to,
			Subscription: subscriptionNone,
			Ask:          true,
		}
	} else {
		ri.Ask = true
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(username, ri); err != nil {
		return err
	}
	r.pushRosterItem(ri)

	// send presence to contact
	p := xml.NewMutablePresenceType(xml.SubscribeType)
	p.SetFrom(r.strm.JID().ToBareJID())
	p.SetTo(to.ToBareJID())
	r.strmManager.SendElement(p, to)
	return nil
}

func (r *Roster) updateRosterItem(rosterItem *entity.RosterItem) (*entity.RosterItem, error) {
	username := r.strm.Username()
	resource := r.strm.Resource()

	jid := rosterItem.JID.ToBareJID()

	switch rosterItem.Subscription {
	case subscriptionRemove:
		log.Infof("removing roster item: %s (%s/%s)", jid, username, resource)
		if err := storage.Instance().DeleteRosterItem(username, jid); err != nil {
			return nil, err
		}
		return rosterItem, nil

	default:
		log.Infof("inserting/updating roster item: %s (%s/%s)", jid, username, resource)

		ri, err := storage.Instance().FetchRosterItem(username, jid)
		if err != nil {
			return nil, err
		}
		if ri != nil {
			// update roster item
			if len(rosterItem.Name) > 0 {
				ri.Name = rosterItem.Name
			}
			ri.Groups = rosterItem.Groups

		} else {
			ri = rosterItem
			ri.Subscription = subscriptionNone
		}
		if err := storage.Instance().InsertOrUpdateRosterItem(username, ri); err != nil {
			return nil, err
		}
		return ri, nil
	}
}

func (r *Roster) pushRosterItem(item *entity.RosterItem) {
	query := xml.NewMutableElementNamespace("query", rosterNamespace)
	query.AppendElement(item.Element())

	userStreams := r.strmManager.UserStreams(r.strm.Username())
	for _, strm := range userStreams {
		if !strm.RequestedRoster() {
			continue
		}
		pushEl := xml.NewMutableIQType(uuid.New(), xml.SetType)
		pushEl.SetTo(strm.JID().ToFullJID())
		pushEl.AppendMutableElement(query)
		strm.SendElement(pushEl)
	}
}
