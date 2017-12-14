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

type SendCallback func(stanza xml.Stanza, sent bool)

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
		authedStrms = removeStreamWithResource(authedStrms, strm.Resource())
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

func (m *StreamManager) send(stanza xml.Stanza, callback SendCallback) {
	toJid := stanza.ToJID()
	recipients := m.authedStrms[toJid.Node()]
	if recipients == nil {
		callback(stanza, false)
		return
	}
	resource := toJid.Resource()
	for _, recipient := range recipients {
		if len(resource) > 0 && recipient.Resource() != resource {
			continue
		}
		recipient.SendElement(stanza)
	}
}

func removeStreamWithResource(strms []*Stream, resource string) []*Stream {
	ret := strms[:0]
	for _, s := range strms {
		if s.Resource() != resource {
			ret = append(ret, s)
		}
	}
	return ret
}
