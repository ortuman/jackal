/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
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
)

const rosterNamespace = "jabber:iq:roster"

type Roster struct {
	queue concurrent.OperationQueue
	strm  Stream

	reqRosterMu sync.RWMutex
	reqRoster   bool
}

func NewRoster(strm Stream) *Roster {
	return &Roster{
		queue: concurrent.OperationQueue{
			QueueSize: 32,
			Timeout:   time.Second,
		},
		strm: strm,
	}
}

func (r *Roster) RequestedRoster() bool {
	r.reqRosterMu.RLock()
	defer r.reqRosterMu.RUnlock()
	return r.reqRoster
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
	if items != nil && len(items) > 0 {
		for _, item := range items {
			q.AppendElement(item.Element())
		}
	}
	result.AppendMutableElement(q)
	r.strm.SendElement(result)

	r.reqRosterMu.Lock()
	r.reqRoster = true
	r.reqRosterMu.Unlock()
}

func (r *Roster) updateRoster(iq *xml.IQ, query *xml.Element) {
	items := query.FindElements("item")
	if len(items) != 1 {
		r.strm.SendElement(iq.BadRequestError())
		return
	}
	ri, err := entity.NewRosterItem(items[0])
	if err != nil {
		r.strm.SendElement(iq.BadRequestError())
		return
	}
	if err := r.updateRosterItem(ri); err != nil {
		log.Error(err)
		r.strm.SendElement(iq.InternalServerError())
		return
	}
	r.strm.PushRosterItem(ri)
	r.strm.SendElement(iq.ResultIQ())
}

func (r *Roster) updateRosterItem(ri *entity.RosterItem) error {
	jid := ri.JID.String()
	switch ri.Subscription {
	case "remove":
		log.Infof("removing roster item: %s (%s/%s)", jid, r.strm.Username(), r.strm.Resource())

		if err := storage.Instance().DeleteRosterItem(r.strm.Username(), ri.JID.ToBareJID()); err != nil {
			return err
		}
	default:
		log.Infof("inserting/updating roster item: %s (%s/%s)", jid, r.strm.Username(), r.strm.Resource())

		if err := storage.Instance().InsertOrUpdateRosterItem(r.strm.Username(), ri); err != nil {
			return err
		}
	}
	return nil
}
