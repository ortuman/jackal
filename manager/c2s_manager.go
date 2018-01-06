/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package manager

import (
	"sync"

	"github.com/ortuman/jackal/log"
)

type availableStreamsReq struct {
	username string
	resultCh chan []C2SStream
}

type C2SManager struct {
	strms       map[string]C2SStream
	authedStrms map[string][]C2SStream

	regCh            chan C2SStream
	unregCh          chan C2SStream
	authCh           chan C2SStream
	availableStrmsCh chan *availableStreamsReq
}

// singleton interface
var (
	instance *C2SManager
	once     sync.Once
)

func C2S() *C2SManager {
	once.Do(func() {
		instance = &C2SManager{
			strms:            make(map[string]C2SStream),
			authedStrms:      make(map[string][]C2SStream),
			regCh:            make(chan C2SStream),
			unregCh:          make(chan C2SStream),
			authCh:           make(chan C2SStream),
			availableStrmsCh: make(chan *availableStreamsReq, 1024),
		}
		go instance.loop()
	})
	return instance
}

func (m *C2SManager) RegisterStream(strm C2SStream) {
	m.regCh <- strm
}

func (m *C2SManager) UnregisterStream(strm C2SStream) {
	m.unregCh <- strm
}

func (m *C2SManager) AuthenticateStream(strm C2SStream) {
	m.authCh <- strm
}

func (m *C2SManager) AvailableStreams(username string) []C2SStream {
	req := &availableStreamsReq{
		username: username,
		resultCh: make(chan []C2SStream),
	}
	m.availableStrmsCh <- req
	return <-req.resultCh
}

func (r *C2SManager) loop() {
	for {
		select {
		case strm := <-r.regCh:
			r.registerStream(strm)
		case strm := <-r.unregCh:
			r.unregisterStream(strm)
		case strm := <-r.authCh:
			r.authenticateStream(strm)
		case req := <-r.availableStrmsCh:
			req.resultCh <- r.availableStreams(req.username)
		}
	}
}

func (r *C2SManager) registerStream(strm C2SStream) {
	log.Infof("registered stream... (id: %s)", strm.ID())
	r.strms[strm.ID()] = strm
}

func (r *C2SManager) unregisterStream(strm C2SStream) {
	log.Infof("unregistered stream... (id: %s)", strm.ID())

	if authedStrms := r.authedStrms[strm.Username()]; authedStrms != nil {
		res := strm.Resource()
		for i := 0; i < len(authedStrms); i++ {
			if res == authedStrms[i].Resource() {
				authedStrms = append(authedStrms[:i], authedStrms[i+1:]...)
				break
			}
		}
		if len(authedStrms) == 0 {
			delete(r.authedStrms, strm.Username())
		}
	}
	delete(r.strms, strm.ID())
}

func (r *C2SManager) authenticateStream(strm C2SStream) {
	log.Infof("authenticated stream... (%s)", strm.Username())

	if authedStrms := r.authedStrms[strm.Username()]; authedStrms != nil {
		r.authedStrms[strm.Username()] = append(authedStrms, strm)
	} else {
		r.authedStrms[strm.Username()] = []C2SStream{strm}
	}
}

func (m *C2SManager) availableStreams(username string) []C2SStream {
	if authedStrms := m.authedStrms[username]; authedStrms != nil {
		return authedStrms
	}
	return []C2SStream{}
}
