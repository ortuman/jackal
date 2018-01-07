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
			Timeout:   time.Second * 10,
		},
		strm: strm,
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
		if err := r.performSubscribed(presence); err != nil {
			log.Error(err)
			return
		}
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
	username := r.strm.Username()
	res := r.strm.Resource()

	userJID := r.strm.JID()
	contactJID := presence.ToJID()

	log.Infof("authorization requested: %v -> %v (%s/%s)", userJID.ToBareJID(), contactJID, username, res)

	ri, err := storage.Instance().FetchRosterItem(userJID.Node(), contactJID.ToBareJID())
	if err != nil {
		return err
	}
	if ri == nil {
		// create roster item if not previously created
		ri = &entity.RosterItem{
			Username:     username,
			JID:          contactJID,
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

	// send presence approval notification to contact
	p := xml.NewMutableElementName("presence")
	p.SetFrom(userJID.ToBareJID())
	p.SetTo(contactJID.ToBareJID())
	p.SetType(xml.SubscribeType)
	p.AppendElements(presence.Elements())

	// archive roster approval notification
	err = storage.Instance().InsertOrUpdateRosterApprovalNotification(username, contactJID.ToBareJID(), p.Copy())
	if err != nil {
		return err
	}
	r.sendElement(p, contactJID)
	return nil
}

func (r *Roster) performSubscribed(presence *xml.Presence) error {
	username := r.strm.Username()
	res := r.strm.Resource()

	userJID := presence.ToJID()
	contactJID := r.strm.JID()

	log.Infof("authorization granted: %v <- %v (%s/%s)", userJID.ToBareJID(), contactJID, username, res)

	// remove approval notification
	if err := storage.Instance().DeleteRosterApprovalNotification(userJID.Node(), contactJID.ToBareJID()); err != nil {
		return err
	}

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

	// update contact's roster item...
	switch contactRosterItem.Subscription {
	case subscriptionFrom:
		contactRosterItem.Subscription = subscriptionBoth
	case subscriptionNone:
		contactRosterItem.Subscription = subscriptionFrom
	}
	if err := storage.Instance().InsertOrUpdateRosterItem(contactRosterItem); err != nil {
		return err
	}
	r.pushRosterItem(contactRosterItem)

	// update user's roster item...
	switch userRosterItem.Subscription {
	case subscriptionTo:
		userRosterItem.Subscription = subscriptionBoth
	case subscriptionNone:
		userRosterItem.Subscription = subscriptionTo
	}
	userRosterItem.Ask = false
	if err := storage.Instance().InsertOrUpdateRosterItem(userRosterItem); err != nil {
		return err
	}

	// send 'subscribed' presence to user...
	p := xml.NewMutableElementName("presence")
	p.SetFrom(contactJID.ToBareJID())
	p.SetTo(userJID.ToBareJID())
	p.SetType(xml.SubscribedType)
	p.AppendElements(presence.Elements())
	r.sendElement(p, userJID)

	// send available presence from all of the contact's available resources to the user
	contactStreams := stream.C2S().AvailableStreams(contactJID.Node())
	for _, contactStream := range contactStreams {
		p := xml.NewMutableElementName("presence")
		p.SetFrom(contactStream.JID().ToFullJID())
		p.SetTo(userJID.ToBareJID())
		r.sendElement(p, userJID)
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

		if err := storage.Instance().DeleteRosterApprovalNotification(username, jid); err != nil {
			return nil, err
		}
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

func (r *Roster) sendElement(element xml.Serializable, to *xml.JID) {
	streams := stream.C2S().AvailableStreams(to.Node())
	for _, strm := range streams {
		strm.SendElement(element)
	}
}

func (r *Roster) pushRosterItem(item *entity.RosterItem) {
	query := xml.NewMutableElementNamespace("query", rosterNamespace)
	query.AppendElement(item.Element())

	streams := stream.C2S().AvailableStreams(r.strm.Username())
	for _, strm := range streams {
		if !strm.RequestedRoster() {
			continue
		}
		pushEl := xml.NewMutableIQType(uuid.New(), xml.SetType)
		pushEl.SetTo(strm.JID().ToFullJID())
		pushEl.AppendMutableElement(query)
		strm.SendElement(pushEl)
	}
}
