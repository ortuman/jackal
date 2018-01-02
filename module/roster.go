/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"time"

	"sync"

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
			if q.ElementsCount() > 0 {
				r.strm.SendElement(iq.BadRequestError())
				return
			}
			r.sendRoster(iq)
		} else if iq.IsSet() {
			r.updateRoster(iq, q)
		}
	})
}

func (r *Roster) sendRoster(iq *xml.IQ) {
	log.Infof("retrieving user roster... (%s/%s)", r.strm.Username(), r.strm.Resource())

	result := iq.ResultIQ()
	query := xml.NewMutableElementNamespace("query", rosterNamespace)

	items, err := storage.Instance().FetchRosterItems(r.strm.Username())
	if err != nil {
		log.Error(err)
		r.strm.SendElement(iq.InternalServerError())
		return
	}
	if items != nil && len(items) > 0 {
		for _, item := range items {
			query.AppendElement(item.Element())
		}
	}
	result.AppendMutableElement(query)
	r.strm.SendElement(result)

	r.reqRosterMu.Lock()
	r.reqRoster = true
	r.reqRosterMu.Unlock()
}

func (r *Roster) updateRoster(iq *xml.IQ, query *xml.Element) {
	items := query.FindElements("item")
	for _, item := range items {
		if ok := r.updateRosterItem(iq, item); !ok {
			return
		}
	}
	r.strm.RosterPush(query)
	r.strm.SendElement(iq.ResultIQ())
}

func (r *Roster) updateRosterItem(iq *xml.IQ, item *xml.Element) bool {
	ri, err := entity.NewRosterItem(item)
	if err != nil {
		r.strm.SendElement(iq.BadRequestError())
		return false
	}
	switch ri.Subscription {
	case "remove":
		if err := storage.Instance().DeleteRosterItem(r.strm.Username(), ri.JID.ToBareJID()); err != nil {
			log.Error(err)
			r.strm.SendElement(iq.InternalServerError())
			return false
		}
	default:
		if err := storage.Instance().InsertOrUpdateRosterItem(r.strm.Username(), ri); err != nil {
			log.Error(err)
			r.strm.SendElement(iq.InternalServerError())
			return false
		}
	}
	return true
}
