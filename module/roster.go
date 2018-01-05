/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
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
	queue concurrent.OperationQueue
	strm  Stream
	once  sync.Once

	requestedMu sync.RWMutex
	requested   bool
}

func NewRoster(stream Stream) *Roster {
	return &Roster{
		queue: concurrent.OperationQueue{
			QueueSize: 32,
			Timeout:   time.Second * 10,
		},
		strm: stream,
	}
}

func (r *Roster) RequestedRoster() bool {
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
	if presence.IsSubscribe() {
		if err := r.performSubscribe(presence); err != nil {
			log.Error(err)
			return
		}
	} else if presence.IsSubscribed() {
		// TODO: Handle 'subscribed' presence
	}
}

func (r *Roster) deliverPendingApprovalNotifications() {
	notifications, err := storage.Instance().FetchRosterApprovalNotifications(r.strm.JID().ToBareJID())
	if err != nil {
		log.Error(err)
		return
	}
	for _, notification := range notifications {
		r.strm.SendElement(notification)
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

	r.requestedMu.Lock()
	r.requested = true
	r.requestedMu.Unlock()
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

func (r *Roster) performSubscribe(presence *xml.Presence) error {
	to := presence.ToJID()

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
			Username:     username,
			JID:          to,
			Subscription: subscriptionNone,
			Ask:          true,
		}
	} else {
		ri.Ask = true
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(ri); err != nil {
		return err
	}
	r.pushRosterItem(ri)

	// send presence to contact
	p := xml.NewMutableElementName("presence")
	p.SetFrom(r.strm.JID().ToBareJID())
	p.SetTo(jid)
	p.SetType(xml.SubscribeType)
	p.AppendElements(presence.Elements())

	// archive roster approval notification
	if err := storage.Instance().InsertOrUpdateRosterApprovalNotification(username, jid, p.Copy()); err != nil {
		return err
	}

	contactStreams := r.strm.UserStreams(to.Node())
	if len(contactStreams) > 0 {
		for _, strm := range contactStreams {
			strm.SendElement(p)
		}
	}
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
			ri = &entity.RosterItem{
				Username:     username,
				JID:          rosterItem.JID,
				Name:         rosterItem.Name,
				Subscription: subscriptionNone,
				Groups:       rosterItem.Groups,
				Ask:          rosterItem.Ask,
			}
		}
		if err := storage.Instance().InsertOrUpdateRosterItem(ri); err != nil {
			return nil, err
		}
		return ri, nil
	}
}

func (r *Roster) pushRosterItem(item *entity.RosterItem) {
	query := xml.NewMutableElementNamespace("query", rosterNamespace)
	query.AppendElement(item.Element())

	userStreams := r.strm.UserStreams(r.strm.Username())
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
