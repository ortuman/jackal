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

type resourceAvailableReq struct {
	resource string
	strm     *C2SStream
	resultCh chan bool
}

type sendRequest struct {
	stanza xml.Stanza
	from   *C2SStream
}

type StreamManager struct {
	strms       map[string]*C2SStream
	authedStrms map[string][]*C2SStream

	regCh      chan *C2SStream
	unregCh    chan *C2SStream
	authCh     chan *C2SStream
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
			strms:       make(map[string]*C2SStream),
			authedStrms: make(map[string][]*C2SStream),
			regCh:       make(chan *C2SStream),
			unregCh:     make(chan *C2SStream),
			authCh:      make(chan *C2SStream),
			resAvailCh:  make(chan *resourceAvailableReq),
			sendCh:      make(chan *sendRequest, 1000),
		}
		go instance.loop()
	})
	return instance
}

func (m *StreamManager) RegisterStream(strm *C2SStream) {
	m.regCh <- strm
}

func (m *StreamManager) UnregisterStream(strm *C2SStream) {
	m.unregCh <- strm
}

func (m *StreamManager) AuthenticateStream(strm *C2SStream) {
	m.authCh <- strm
}

func (m *StreamManager) IsResourceAvailable(resource string, strm *C2SStream) bool {
	req := &resourceAvailableReq{
		resource: resource,
		strm:     strm,
		resultCh: make(chan bool),
	}
	m.resAvailCh <- req
	return <-req.resultCh
}

func (m *StreamManager) Send(stanza xml.Stanza, from *C2SStream) {
	m.sendCh <- &sendRequest{
		stanza: stanza,
		from:   from,
	}
}

func (m *StreamManager) loop() {
	for {
		select {
		case req := <-m.sendCh:
			m.send(req.stanza, req.from)
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

func (m *StreamManager) registerStream(strm *C2SStream) {
	log.Infof("registered stream... (id: %s)", strm.ID())
	m.strms[strm.ID()] = strm
}

func (m *StreamManager) unregisterStream(strm *C2SStream) {
	log.Infof("unregistered stream... (id: %s)", strm.ID())

	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		authedStrms = removeStreamWithResource(authedStrms, strm.Resource())
		if len(authedStrms) == 0 {
			delete(m.authedStrms, strm.Username())
		}
	}
	delete(m.strms, strm.ID())
}

func (m *StreamManager) authenticateStream(strm *C2SStream) {
	log.Infof("authenticated stream... (username: %s)", strm.Username())

	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		m.authedStrms[strm.Username()] = append(authedStrms, strm)
	} else {
		m.authedStrms[strm.Username()] = []*C2SStream{strm}
	}
}

func (m *StreamManager) isResourceAvailable(resource string, strm *C2SStream) bool {
	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		for _, authedStrm := range authedStrms {
			if authedStrm.Resource() == resource {
				return false
			}
		}
	}
	return true
}

func (m *StreamManager) send(stanza xml.Stanza, from *C2SStream) {
}

func removeStreamWithResource(strms []*C2SStream, resource string) []*C2SStream {
	ret := strms[:0]
	for _, s := range strms {
		if s.Resource() != resource {
			ret = append(ret, s)
		}
	}
	return ret
}
