/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import (
	"errors"
	"sync"
	"time"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// InStream represents a generic incoming stream.
type InStream interface {
	ID() string
	Disconnect(err error)
}

// InOutStream represents a generic incoming/outgoing stream.
type InOutStream interface {
	InStream
	SendElement(elem xmpp.XElement)
}

// C2S represents a client-to-server XMPP stream.
type C2S interface {
	InOutStream

	Context() map[string]interface{}

	SetString(key string, value string)
	GetString(key string) string

	SetInt(key string, value int)
	GetInt(key string) int

	SetFloat(key string, value float64)
	GetFloat(key string) float64

	SetBool(key string, value bool)
	GetBool(key string) bool

	Username() string
	Domain() string
	Resource() string

	JID() *jid.JID

	IsSecured() bool
	IsAuthenticated() bool

	Presence() *xmpp.Presence
}

// S2SIn represents an incoming server-to-server XMPP stream.
type S2SIn interface {
	InStream
}

// S2SOut represents an outgoing server-to-server XMPP stream.
type S2SOut interface {
	InOutStream
}

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
	contextMu       sync.RWMutex
	context         map[string]interface{}
	elemCh          chan xmpp.XElement
	actorCh         chan func()
	discCh          chan error
}

// NewMockC2S returns a new mocked stream instance.
func NewMockC2S(id string, jid *jid.JID) *MockC2S {
	stm := &MockC2S{
		id:      id,
		context: make(map[string]interface{}),
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

// Context returns a copy of the stream associated context.
func (m *MockC2S) Context() map[string]interface{} {
	ret := make(map[string]interface{})
	m.contextMu.RLock()
	for k, v := range m.context {
		ret[k] = v
	}
	m.contextMu.RUnlock()
	return ret
}

// SetString associates a string context value to a key.
func (m *MockC2S) SetString(key string, value string) {
	m.setContextValue(key, value)
}

// GetString returns the context value associated with the key as a string.
func (m *MockC2S) GetString(key string) string {
	var ret string
	m.contextMu.RLock()
	defer m.contextMu.RUnlock()
	if s, ok := m.context[key].(string); ok {
		ret = s
	}
	return ret
}

// SetInt associates an integer context value to a key.
func (m *MockC2S) SetInt(key string, value int) {
	m.setContextValue(key, value)
}

// GetInt returns the context value associated with the key as an integer.
func (m *MockC2S) GetInt(key string) int {
	var ret int
	m.contextMu.RLock()
	defer m.contextMu.RUnlock()
	if i, ok := m.context[key].(int); ok {
		ret = i
	}
	return ret
}

// SetInt associates a float context value to a key.
func (m *MockC2S) SetFloat(key string, value float64) {
	m.setContextValue(key, value)
}

// GetFloat returns the context value associated with the key as a float64.
func (m *MockC2S) GetFloat(key string) float64 {
	var ret float64
	m.contextMu.RLock()
	defer m.contextMu.RUnlock()
	if f, ok := m.context[key].(float64); ok {
		ret = f
	}
	return ret
}

// SetBool associates a boolean context value to a key.
func (m *MockC2S) SetBool(key string, value bool) {
	m.setContextValue(key, value)
}

// GetBool returns the context value associated with the key as a boolean.
func (m *MockC2S) GetBool(key string) bool {
	var ret bool
	m.contextMu.RLock()
	defer m.contextMu.RUnlock()
	if b, ok := m.context[key].(bool); ok {
		ret = b
	}
	return ret
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
func (m *MockC2S) SendElement(elem xmpp.XElement) {
	m.actorCh <- func() {
		m.sendElement(elem)
	}
}

// Disconnect disconnects mocked stream.
func (m *MockC2S) Disconnect(err error) {
	waitCh := make(chan struct{})
	m.actorCh <- func() {
		m.disconnect(err)
		close(waitCh)
	}
	<-waitCh
}

// FetchElement waits until a new XML element is sent to
// the mocked stream and returns it.
func (m *MockC2S) FetchElement() xmpp.XElement {
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

func (m *MockC2S) setContextValue(key string, value interface{}) {
	m.contextMu.Lock()
	defer m.contextMu.Unlock()
	m.context[key] = value
}
