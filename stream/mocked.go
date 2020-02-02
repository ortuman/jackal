/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// MockC2S represents a mocked c2s stream.
type MockC2S struct {
	id              string
	mu              sync.RWMutex
	isSecured       bool
	isAuthenticated bool
	isCompressed    bool
	isDisconnected  bool
	jid             *jid.JID
	presence        *xmpp.Presence
	elemCh          chan xmpp.XElement
	actorCh         chan func()
	discCh          chan error
	ctx             context.Context
}

// NewMockC2S returns a new mocked stream instance.
func NewMockC2S(id string, jid *jid.JID) *MockC2S {
	stm := &MockC2S{
		id:      id,
		ctx:     context.Background(),
		elemCh:  make(chan xmpp.XElement, 16),
		actorCh: make(chan func(), 64),
		discCh:  make(chan error, 1),
	}
	stm.SetJID(jid)
	go stm.actorLoop()
	return stm
}

// ID returns mocked stream identifier.
func (m *MockC2S) ID() string {
	return m.id
}

func (m *MockC2S) Context() context.Context {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ctx
}

func (m *MockC2S) Value(key interface{}) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ctx.Value(key)
}

func (m *MockC2S) SetValue(key, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx = context.WithValue(m.ctx, key, value)
}

// Username returns current mocked stream username.
func (m *MockC2S) Username() string {
	return m.JID().Node()
}

// Domain returns current mocked stream domain.
func (m *MockC2S) Domain() string {
	return m.JID().Domain()
}

// Resource returns current mocked stream resource.
func (m *MockC2S) Resource() string {
	return m.JID().Resource()
}

// SetJID sets the mocked stream JID value.
func (m *MockC2S) SetJID(jid *jid.JID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jid = jid
}

// JID returns current user JID.
func (m *MockC2S) JID() *jid.JID {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.jid
}

// SetSecured sets whether or not the a mocked stream
// has been secured.
func (m *MockC2S) SetSecured(secured bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isSecured = secured
}

// IsSecured returns whether or not the mocked stream
// has been secured.
func (m *MockC2S) IsSecured() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isSecured
}

// SetAuthenticated sets whether or not the a mocked stream
// has been authenticated.
func (m *MockC2S) SetAuthenticated(authenticated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isAuthenticated = authenticated
}

// IsAuthenticated returns whether or not the mocked stream
// has successfully authenticated.
func (m *MockC2S) IsAuthenticated() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isAuthenticated
}

// IsDisconnected returns whether or not the mocked stream has been disconnected.
func (m *MockC2S) IsDisconnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isDisconnected
}

// SetPresence sets the mocked stream last received
// presence element.
func (m *MockC2S) SetPresence(presence *xmpp.Presence) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.presence = presence
}

// Presence returns last sent presence element.
func (m *MockC2S) Presence() *xmpp.Presence {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.presence
}

// SendElement sends the given XML element.
func (m *MockC2S) SendElement(_ context.Context, elem xmpp.XElement) {
	m.actorCh <- func() {
		m.sendElement(elem)
	}
}

// Disconnect disconnects mocked stream.
func (m *MockC2S) Disconnect(_ context.Context, err error) {
	waitCh := make(chan struct{})
	m.actorCh <- func() {
		m.disconnect(err)
		close(waitCh)
	}
	<-waitCh
}

// ReceiveElement waits until a new XML element is sent to
// the mocked stream and returns it.
func (m *MockC2S) ReceiveElement() xmpp.XElement {
	select {
	case e := <-m.elemCh:
		return e
	case <-time.After(time.Second * 5):
		return &xmpp.Element{}
	}
}

// WaitDisconnection waits until the mocked stream disconnects.
func (m *MockC2S) WaitDisconnection() error {
	select {
	case err := <-m.discCh:
		return err
	case <-time.After(time.Second * 5):
		return errors.New("operation timed out")
	}
}

func (m *MockC2S) actorLoop() {
	for {
		select {
		case f := <-m.actorCh:
			f()
		case <-m.discCh:
			return
		}
	}
}

func (m *MockC2S) sendElement(elem xmpp.XElement) {
	select {
	case m.elemCh <- elem:
		return
	default:
		break
	}
}

func (m *MockC2S) disconnect(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.isDisconnected {
		m.discCh <- err
		m.isDisconnected = true
	}
}
