/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/xml"
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

type sendRequest struct {
	stanza   xml.Stanza
	callback SendCallback
}

type StreamManager struct {
	strms       map[string]*Stream
	authedStrms map[string][]*Stream

	regCh      chan *Stream
	unregCh    chan *Stream
	authCh     chan *Stream
	resAvailCh chan *resourceAvailableReq
	sendCh     chan *sendRequest
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
			resAvailCh:  make(chan *resourceAvailableReq),
			sendCh:      make(chan *sendRequest, 1000),
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
	m.sendCh <- &sendRequest{
		stanza:   stanza,
		callback: callback,
	}
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
	log.Infof("authenticated stream... (username: %s)", strm.Username())

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
			// send to highest priority
			break

		case *xml.Presence:
			// broadcast presence
			for _, strm := range recipients {
				strm.SendElement(stanza)
			}
		}
	}
	if sendCallback != nil {
		sendCallback.Sent(stanza)
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
