/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"sync"

	"github.com/ortuman/jackal/concurrent"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/xml"
)

type StreamManager struct {
	concurrent.ExecutorQueue
	strms       map[string]*Stream
	authedStrms map[string][]*Stream
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
		}
	})
	return instance
}

func (m *StreamManager) RegisterStream(strm *Stream) {
	m.Sync(func() {
		log.Infof("registered stream... (id: %s)", strm.ID())
		m.strms[strm.ID()] = strm
	})
}

func (m *StreamManager) UnregisterStream(strm *Stream) {
	m.Sync(func() {
		log.Infof("unregistered stream... (id: %s)", strm.ID())
		if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
			authedStrms = removeStreamWithResource(authedStrms, strm.Resource())
			if len(authedStrms) == 0 {
				delete(m.authedStrms, strm.Username())
			}
		}
	})
}

func (m *StreamManager) AuthenticateStream(strm *Stream) {
	m.Sync(func() {
		log.Infof("authenticated stream... (username: %s)", strm.Username())
		if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
			m.authedStrms[strm.Username()] = append(authedStrms, strm)
		} else {
			m.authedStrms[strm.Username()] = []*Stream{strm}
		}
	})
}

func (m *StreamManager) IsResourceAvailableForStream(resource string, strm *Stream) bool {
	ch := make(chan bool)
	m.Async(func() {
		if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
			for _, authedStrm := range authedStrms {
				if authedStrm.Resource() == resource {
					ch <- false
					return
				}
			}
			ch <- true
		}
	})
	return <-ch
}

func (m *StreamManager) Send(stanza xml.Stanza, from *Stream) {
	m.Async(func() {
	})
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
