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

type C2SStream interface {
	ID() string

	Username() string
	Domain() string
	Resource() string

	JID() *xml.JID

	Secured() bool
	Authenticated() bool
	Compressed() bool

	Priority() int8

	Active() bool
	Available() bool

	RequestedRoster() bool

	SendElement(element xml.Serializable)
	Disconnect(err error)
}

type availableStreamsReq struct {
	username string
	resultCh chan []C2SStream
}

type C2SManager struct {
	sync.RWMutex
	strms       map[string]C2SStream
	authedStrms map[string][]C2SStream
}

// singleton interface
var (
	instance *C2SManager
	once     sync.Once
)

func C2S() *C2SManager {
	once.Do(func() {
		instance = &C2SManager{
			strms:       make(map[string]C2SStream),
			authedStrms: make(map[string][]C2SStream),
		}
	})
	return instance
}

func (m *C2SManager) RegisterStream(strm C2SStream) {
	m.Lock()
	defer m.Unlock()

	log.Infof("registered stream... (id: %s)", strm.ID())

	m.strms[strm.ID()] = strm
}

func (m *C2SManager) UnregisterStream(strm C2SStream) {
	m.Lock()
	defer m.Unlock()

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

func (m *C2SManager) AuthenticateStream(strm C2SStream) {
	m.Lock()
	defer m.Unlock()

	log.Infof("authenticated stream... (%s)", strm.Username())

	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		m.authedStrms[strm.Username()] = append(authedStrms, strm)
	} else {
		m.authedStrms[strm.Username()] = []C2SStream{strm}
	}
}

func (m *C2SManager) AvailableStreams(username string) []C2SStream {
	m.RLock()
	defer m.RUnlock()
	return m.authedStrms[username]
}
