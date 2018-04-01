/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/xml"
)

// Stream represents a client-to-server XMPP stream.
type Stream interface {
	ID() string

	Username() string
	Domain() string
	Resource() string

	JID() *xml.JID

	Priority() int8

	SendElement(element xml.XElement)
	Disconnect(err error)

	IsSecured() bool
	IsAuthenticated() bool
	IsCompressed() bool

	PresenceElements() []xml.XElement

	IsRosterRequested() bool
}

// Manager manages the sessions associated with an account.
type Manager struct {
	cfg         *config.C2S
	lock        sync.RWMutex
	strms       map[string]Stream
	authedStrms map[string][]Stream
}

// singleton interface
var (
	inst        *Manager
	instMu      sync.RWMutex
	initialized uint32
)

// Initialize initializes the c2s session manager.
func Initialize(cfg *config.C2S) {
	if atomic.CompareAndSwapUint32(&initialized, 0, 1) {
		instMu.Lock()
		defer instMu.Unlock()

		inst = &Manager{
			cfg:         cfg,
			strms:       make(map[string]Stream),
			authedStrms: make(map[string][]Stream),
		}
	}
}

// Instance returns the c2s session manager instance.
func Instance() *Manager {
	instMu.RLock()
	defer instMu.RUnlock()

	if inst == nil {
		log.Fatalf("c2s manager not initialized")
	}
	return inst
}

// Shutdown shuts down c2s manager system.
// This method should be used only for testing purposes.
func Shutdown() {
	if atomic.CompareAndSwapUint32(&initialized, 1, 0) {
		instMu.Lock()
		defer instMu.Unlock()
		inst = nil
	}
}

// DefaultLocalDomain returns default local domain.
func (m *Manager) DefaultLocalDomain() string {
	return m.cfg.Domains[0]
}

// IsLocalDomain returns true if domain is a local server domain.
func (m *Manager) IsLocalDomain(domain string) bool {
	for _, localDomain := range m.cfg.Domains {
		if localDomain == domain {
			return true
		}
	}
	return false
}

// RegisterStream registers the specified client stream.
// An error will be returned in case the stream has been previously registered.
func (m *Manager) RegisterStream(strm Stream) error {
	if !m.IsLocalDomain(strm.Domain()) {
		return fmt.Errorf("invalid domain: %s", strm.Domain())
	}
	m.lock.Lock()
	_, ok := m.strms[strm.ID()]
	if ok {
		m.lock.Unlock()
		return fmt.Errorf("stream already registered: %s", strm.ID())
	}
	m.strms[strm.ID()] = strm
	m.lock.Unlock()
	log.Infof("registered stream... (id: %s)", strm.ID())
	return nil
}

// UnregisterStream unregisters the specified client stream removing
// associated resource from the manager.
// An error will be returned in case the stream has not been previously registered.
func (m *Manager) UnregisterStream(strm Stream) error {
	m.lock.Lock()
	_, ok := m.strms[strm.ID()]
	if !ok {
		m.lock.Unlock()
		return fmt.Errorf("stream not found: %s", strm.ID())
	}
	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		res := strm.Resource()
		for i := 0; i < len(authedStrms); i++ {
			if res == authedStrms[i].Resource() {
				authedStrms = append(authedStrms[:i], authedStrms[i+1:]...)
				break
			}
		}
		if len(authedStrms) > 0 {
			m.authedStrms[strm.Username()] = authedStrms
		} else {
			delete(m.authedStrms, strm.Username())
		}
	}
	delete(m.strms, strm.ID())
	m.lock.Unlock()
	log.Infof("unregistered stream... (id: %s)", strm.ID())
	return nil
}

// AuthenticateStream sets a previously registered stream as authenticated.
// An error will be returned in case no assigned resource is found.
func (m *Manager) AuthenticateStream(strm Stream) error {
	if len(strm.Resource()) == 0 {
		return fmt.Errorf("resource not yet assigned: %s", strm.ID())
	}
	m.lock.Lock()
	if authedStrms := m.authedStrms[strm.Username()]; authedStrms != nil {
		m.authedStrms[strm.Username()] = append(authedStrms, strm)
	} else {
		m.authedStrms[strm.Username()] = []Stream{strm}
	}
	m.lock.Unlock()
	log.Infof("authenticated stream... (%s/%s)", strm.Username(), strm.Resource())
	return nil
}

// AvailableStreams returns every authenticated stream associated with an account.
func (m *Manager) AvailableStreams(username string) []Stream {
	m.lock.RLock()
	res := m.authedStrms[username]
	m.lock.RUnlock()
	return res
}
