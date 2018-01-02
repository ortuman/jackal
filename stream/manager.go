/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/entity"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

type SendCallback interface {
	Sent(xml.Stanza)
	NotAuthenticated(xml.Stanza)
	ResourceNotFound(xml.Stanza)
}

type resourceAvailableReq struct {
	resource string
	strm     *Stream
	resultCh chan bool
}

type sendReq struct {
	stanza   xml.Stanza
	callback SendCallback
}

type rosterPushReq struct {
	username string
	item     *entity.RosterItem
	doneCh   chan struct{}
}

type StreamManager struct {
	strms       map[string]*Stream
	authedStrms map[string][]*Stream

	regCh        chan *Stream
	unregCh      chan *Stream
	authCh       chan *Stream
	rosterPushCh chan *rosterPushReq
	resAvailCh   chan *resourceAvailableReq
	sendCh       chan *sendReq
}

// singleton interface
var (
	instance *StreamManager
	once     sync.Once
)

func Manager() *StreamManager {
	once.Do(func() {
		instance = &StreamManager{
			strms:        make(map[string]*Stream),
			authedStrms:  make(map[string][]*Stream),
			regCh:        make(chan *Stream, 32),
			unregCh:      make(chan *Stream, 32),
			authCh:       make(chan *Stream, 32),
			rosterPushCh: make(chan *rosterPushReq, 32),
			resAvailCh:   make(chan *resourceAvailableReq, 32),
			sendCh:       make(chan *sendReq, 1000),
		}
		go instance.loop()
	})
	return instance
}

func (m *StreamManager) RegisterStream(strm *Stream) {
	m.regCh <- strm
}

func (m *StreamManager) UnregisterStream(strm *Stream) {
	m.unregCh <- strm
}

func (m *StreamManager) AuthenticateStream(strm *Stream) {
	m.authCh <- strm
}

func (m *StreamManager) IsResourceAvailable(resource string, strm *Stream) bool {
	req := &resourceAvailableReq{
		resource: resource,
		strm:     strm,
		resultCh: make(chan bool),
	}
	m.resAvailCh <- req
	return <-req.resultCh
}

func (m *StreamManager) Send(stanza xml.Stanza, callback SendCallback) {
	m.sendCh <- &sendReq{
		stanza:   stanza,
		callback: callback,
	}
}

func (m *StreamManager) PushRosterItem(item *entity.RosterItem, username string) {
	req := &rosterPushReq{
		username: username,
		item:     item,
		doneCh:   make(chan struct{}),
	}
	m.rosterPushCh <- req
	<-req.doneCh
}

func (m *StreamManager) loop() {
	for {
		select {
		case req := <-m.sendCh:
			m.send(req.stanza, req.callback)
		case strm := <-m.regCh:
			m.registerStream(strm)
		case strm := <-m.unregCh:
			m.unregisterStream(strm)
		case strm := <-m.authCh:
			m.authenticateStream(strm)
		case req := <-m.rosterPushCh:
			m.pushRosterItem(req.item, req.username, req.doneCh)
		case req := <-m.resAvailCh:
			req.resultCh <- m.isResourceAvailable(req.resource, req.strm)
		}
	}
}

func (m *StreamManager) registerStream(strm *Stream) {
	log.Infof("registered stream... (id: %s)", strm.ID())
	m.strms[strm.ID()] = strm
}

func (m *StreamManager) unregisterStream(strm *Stream) {
	log.Infof("unregistered stream... (id: %s)", strm.ID())

	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		authedStrms = filterStreams(authedStrms, func(s *Stream) bool {
			return s.Resource() != strm.Resource()
		})
		if len(authedStrms) == 0 {
			delete(m.authedStrms, strm.Username())
		}
	}
	delete(m.strms, strm.ID())
}

func (m *StreamManager) authenticateStream(strm *Stream) {
	log.Infof("authenticated stream... (%s)", strm.Username())

	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		m.authedStrms[strm.Username()] = append(authedStrms, strm)
	} else {
		m.authedStrms[strm.Username()] = []*Stream{strm}
	}
}

func (m *StreamManager) isResourceAvailable(resource string, strm *Stream) bool {
	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		for _, authedStrm := range authedStrms {
			if authedStrm.Resource() == resource {
				return false
			}
		}
	}
	return true
}

func (m *StreamManager) send(stanza xml.Stanza, sendCallback SendCallback) {
	toJid := stanza.ToJID()
	recipients := m.authedStrms[toJid.Node()]
	if recipients == nil {
		if sendCallback != nil {
			sendCallback.NotAuthenticated(stanza)
		}
		return
	}
	if toJid.IsFull() {
		recipients = filterStreams(recipients, func(s *Stream) bool {
			return s.Resource() == toJid.Resource()
		})
		if len(recipients) == 0 {
			if sendCallback != nil {
				sendCallback.ResourceNotFound(stanza)
			}
			return
		}
		recipients[0].SendElement(stanza)

	} else {
		switch stanza.(type) {
		case *xml.Message:
			// send to highest priority stream
			if strm := highestPriorityStream(recipients); strm != nil {
				strm.SendElement(stanza)
				goto done
			}
		}
		// broadcast to all streams
		for _, strm := range recipients {
			strm.SendElement(stanza)
		}
	}

done:
	if sendCallback != nil {
		sendCallback.Sent(stanza)
	}
}

func (m *StreamManager) pushRosterItem(item *entity.RosterItem, username string, doneCh chan struct{}) {
	authedStrms := m.authedStrms[username]
	if authedStrms == nil {
		return
	}
	query := xml.NewMutableElementNamespace("query", "jabber:iq:roster")
	query.AppendElement(item.Element())
	for _, authedStrm := range authedStrms {
		if authedStrm.RequestedRoster() {
			pushEl := xml.NewMutableIQType(uuid.New(), xml.SetType)
			pushEl.SetTo(authedStrm.JID().ToFullJID())
			pushEl.AppendMutableElement(query)
			authedStrm.SendElement(pushEl)
		}
	}
	close(doneCh)
}

func filterStreams(strms []*Stream, include func(*Stream) bool) []*Stream {
	length := len(strms)
	res := make([]*Stream, 0, length)
	for _, strm := range strms {
		if include(strm) {
			res = append(res, strm)
		}
	}
	return res
}

func highestPriorityStream(strms []*Stream) *Stream {
	var highestPriority int8 = 0
	var strm *Stream
	for _, s := range strms {
		if s.Priority() > highestPriority {
			strm = s
		}
	}
	return strm
}
