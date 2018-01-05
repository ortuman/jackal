/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import (
	"sync"

	"github.com/ortuman/jackal/log"
)

type userStreamsReq struct {
	username string
	resultCh chan []*Stream
}

type StreamManager struct {
	strms       map[string]*Stream
	authedStrms map[string][]*Stream

	regCh       chan *Stream
	unregCh     chan *Stream
	authCh      chan *Stream
	userStrmsCh chan *userStreamsReq
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
			userStrmsCh: make(chan *userStreamsReq, 1024),
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

func (m *StreamManager) loop() {
	for {
		select {
		case strm := <-m.regCh:
			m.registerStream(strm)
		case strm := <-m.unregCh:
			m.unregisterStream(strm)
		case strm := <-m.authCh:
			m.authenticateStream(strm)
		case req := <-m.userStrmsCh:
			req.resultCh <- m.userStreams(req.username)
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
		res := strm.Resource()
		for i := 0; i < len(authedStrms); i++ {
			if res == authedStrms[i].Resource() {
				authedStrms = append(authedStrms[:i], authedStrms[i+1:]...)
				break
			}
		}
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
