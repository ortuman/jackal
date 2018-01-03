/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import (
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/xml"
)

type SendCallback interface {
	Sent(serializable xml.Serializable, to *xml.JID)
	NotAuthenticated(serializable xml.Serializable, to *xml.JID)
	ResourceNotFound(serializable xml.Serializable, to *xml.JID)
}

type resourceAvailableReq struct {
	resource string
	username string
	resultCh chan bool
}

type userStreamsReq struct {
	username string
	resultCh chan []*Stream
}

type sendReq struct {
	serializable xml.Serializable
	to           *xml.JID
	callback     SendCallback
}

type StreamManager struct {
	strms       map[string]*Stream
	authedStrms map[string][]*Stream

	regCh       chan *Stream
	unregCh     chan *Stream
	authCh      chan *Stream
	userStrmsCh chan *userStreamsReq
	resAvailCh  chan *resourceAvailableReq
	sendCh      chan *sendReq
}

// singleton interface
var (
	instance *StreamManager
	once     sync.Once
)

func Manager() *StreamManager {
	once.Do(func() {
		instance = &StreamManager{
			strms:       make(map[string]*Stream),
			authedStrms: make(map[string][]*Stream),
			regCh:       make(chan *Stream),
			unregCh:     make(chan *Stream),
			authCh:      make(chan *Stream),
			userStrmsCh: make(chan *userStreamsReq),
			resAvailCh:  make(chan *resourceAvailableReq),
			sendCh:      make(chan *sendReq, 1000),
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

func (m *StreamManager) UserStreams(username string) []*Stream {
	req := &userStreamsReq{
		username: username,
		resultCh: make(chan []*Stream),
	}
	m.userStrmsCh <- req
	return <-req.resultCh
}

func (m *StreamManager) ResourceAvailable(resource string, strm *Stream) bool {
	req := &resourceAvailableReq{
		resource: resource,
		username: strm.Username(),
		resultCh: make(chan bool),
	}
	m.resAvailCh <- req
	return <-req.resultCh
}

func (m *StreamManager) SendElement(serializable xml.Serializable, to *xml.JID, callback SendCallback) {
	m.sendCh <- &sendReq{
		serializable: serializable,
		to:           to,
		callback:     callback,
	}
}

func (m *StreamManager) loop() {
	for {
		select {
		case req := <-m.sendCh:
			m.send(req.serializable, req.to, req.callback)
		case strm := <-m.regCh:
			m.registerStream(strm)
		case strm := <-m.unregCh:
			m.unregisterStream(strm)
		case strm := <-m.authCh:
			m.authenticateStream(strm)
		case req := <-m.userStrmsCh:
			req.resultCh <- m.userStreams(req.username)
		case req := <-m.resAvailCh:
			req.resultCh <- m.isResourceAvailable(req.resource, req.username)
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

func (m *StreamManager) userStreams(username string) []*Stream {
	if authedStrms := m.authedStrms[username]; authedStrms != nil {
		return authedStrms
	}
	return []*Stream{}
}

func (m *StreamManager) isResourceAvailable(resource string, username string) bool {
	if authedStrms := m.authedStrms[username]; authedStrms != nil {
		for _, authedStrm := range authedStrms {
			if authedStrm.Resource() == resource {
				return false
			}
		}
	}
	return true
}

func (m *StreamManager) send(serializable xml.Serializable, to *xml.JID, sendCallback SendCallback) {
	recipients := m.authedStrms[to.Node()]
	if recipients == nil {
		if sendCallback != nil {
			sendCallback.NotAuthenticated(serializable, to)
		}
		return
	}
	if to.IsFull() {
		recipients = filterStreams(recipients, func(s *Stream) bool {
			return s.Resource() == to.Resource()
		})
		if len(recipients) == 0 {
			if sendCallback != nil {
				sendCallback.ResourceNotFound(serializable, to)
			}
			return
		}
		recipients[0].SendElement(serializable)

	} else {
		switch serializable.(type) {
		case *xml.Message:
			// send to highest priority stream
			if strm := highestPriorityStream(recipients); strm != nil {
				strm.SendElement(serializable)
				goto done
			}
		}
		// broadcast to all streams
		for _, strm := range recipients {
			strm.SendElement(serializable)
		}
	}

done:
	if sendCallback != nil {
		sendCallback.Sent(serializable, to)
	}
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
