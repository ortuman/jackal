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

type StreamManager struct {
	sync.Mutex
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
	m.Lock()
	defer m.Unlock()
	log.Infof("registered stream... (id: %s)", strm.ID())
	m.strms[strm.ID()] = strm
}

func (m *StreamManager) UnregisterStream(strm *Stream) {
	m.Lock()
	defer m.Unlock()
	log.Infof("unregistered stream... (id: %s)", strm.ID())
	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		authedStrms = removeStreamWithResource(authedStrms, strm.Resource())
		if len(authedStrms) == 0 {
			delete(m.authedStrms, strm.Username())
		}
	}
}

func (m *StreamManager) AuthenticateStream(strm *Stream) {
	m.Lock()
	defer m.Unlock()
	log.Infof("authenticated stream... (username: %s)", strm.Username())
	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		m.authedStrms[strm.Username()] = append(authedStrms, strm)
	} else {
		m.authedStrms[strm.Username()] = []*Stream{strm}
	}
}

func (m *StreamManager) IsResourceAvailableForStream(resource string, strm *Stream) bool {
	m.Lock()
	defer m.Unlock()
	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		for _, authedStrm := range authedStrms {
			if authedStrm.Resource() == resource {
				return false
			}
		}
	}
	return true
}

func (m *StreamManager) Send(stanza xml.Stanza, from *Stream) {
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
