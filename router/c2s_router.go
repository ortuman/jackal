/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"sync"

	"github.com/ortuman/jackal/log"
)

type userStreamsReq struct {
	username string
	resultCh chan []Stream
}

type C2SRouter struct {
	strms       map[string]Stream
	authedStrms map[string][]Stream

	regCh       chan Stream
	unregCh     chan Stream
	authCh      chan Stream
	userStrmsCh chan *userStreamsReq
}

// singleton interface
var (
	instance *C2SRouter
	once     sync.Once
)

func C2S() *C2SRouter {
	once.Do(func() {
		instance = &C2SRouter{
			strms:       make(map[string]Stream),
			authedStrms: make(map[string][]Stream),
			regCh:       make(chan Stream),
			unregCh:     make(chan Stream),
			authCh:      make(chan Stream),
			userStrmsCh: make(chan *userStreamsReq, 1024),
		}
		go instance.loop()
	})
	return instance
}

func (r *C2SRouter) RegisterStream(strm Stream) {
	r.regCh <- strm
}

func (r *C2SRouter) UnregisterStream(strm Stream) {
	r.unregCh <- strm
}

func (r *C2SRouter) AuthenticateStream(strm Stream) {
	r.authCh <- strm
}

func (r *C2SRouter) UserStreams(username string) []Stream {
	req := &userStreamsReq{
		username: username,
		resultCh: make(chan []Stream),
	}
	r.userStrmsCh <- req
	return <-req.resultCh
}

func (r *C2SRouter) loop() {
	for {
		select {
		case strm := <-r.regCh:
			r.registerStream(strm)
		case strm := <-r.unregCh:
			r.unregisterStream(strm)
		case strm := <-r.authCh:
			r.authenticateStream(strm)
		case req := <-r.userStrmsCh:
			req.resultCh <- r.userStreams(req.username)
		}
	}
}

func (r *C2SRouter) registerStream(strm Stream) {
	log.Infof("registered stream... (id: %s)", strm.ID())
	r.strms[strm.ID()] = strm
}

func (r *C2SRouter) unregisterStream(strm Stream) {
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

func (r *C2SRouter) authenticateStream(strm Stream) {
	log.Infof("authenticated stream... (%s)", strm.Username())

	if authedStrms := r.authedStrms[strm.Username()]; authedStrms != nil {
		r.authedStrms[strm.Username()] = append(authedStrms, strm)
	} else {
		r.authedStrms[strm.Username()] = []Stream{strm}
	}
}

func (m *C2SRouter) userStreams(username string) []Stream {
	if authedStrms := m.authedStrms[username]; authedStrms != nil {
		return authedStrms
	}
	return []Stream{}
}
