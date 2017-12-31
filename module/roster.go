/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package module

import (
	"time"

	"github.com/ortuman/jackal/concurrent"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/xml"
)

const rosterNamespace = "jabber:iq:roster"

type Roster struct {
	queue concurrent.OperationQueue
	strm  Stream
}

func NewRoster(strm Stream) *Roster {
	return &Roster{
		queue: concurrent.OperationQueue{
			QueueSize: 16,
			Timeout:   time.Second,
		},
		strm: strm,
	}
}

func (r *Roster) AssociatedNamespaces() []string {
	return []string{}
}

func (r *Roster) MatchesIQ(iq *xml.IQ) bool {
	return iq.FindElementNamespace("query", rosterNamespace) != nil
}

func (r *Roster) ProcessIQ(iq *xml.IQ) {
	r.queue.Async(func() {
		r.sendUserRoster(iq)
	})
}

func (r *Roster) sendUserRoster(iq *xml.IQ) {
	log.Infof("retrieving user roster... (%s/%s)", r.strm.Username(), r.strm.Resource())

	result := iq.ResultIQ()
	query := xml.NewMutableElementNamespace("query", rosterNamespace)
	result.AppendMutableElement(query)
	r.strm.SendElement(result)
}
