/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"sync"

	"github.com/ortuman/jackal/xml"
)

// MockStream represents a mocked c2s stream.
type MockStream struct {
	mu               sync.RWMutex
	id               string
	username         string
	domain           string
	resource         string
	jid              *xml.JID
	priority         int8
	disconnected     bool
	secured          bool
	authenticated    bool
	compressed       bool
	rosterRequested  bool
	presenceElements []xml.XElement
	elemCh           chan xml.XElement
	discCh           chan error
}

// NewMockStream returns a new mocked stream instance.
func NewMockStream(id string, jid *xml.JID) *MockStream {
	strm := &MockStream{}
	strm.id = id
	strm.jid = jid
	strm.username = jid.Node()
	strm.domain = jid.Domain()
	strm.resource = jid.Resource()
	strm.elemCh = make(chan xml.XElement, 16)
	strm.discCh = make(chan error, 1)
	return strm
}

// ID returns mocked stream identifier.
func (m *MockStream) ID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.id
}

// SetID sets mocked stream identifier.
func (m *MockStream) SetID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.id = id
}

// Username returns current mocked stream username.
func (m *MockStream) Username() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.username
}

// SetUsername sets the mocked stream username value.
func (m *MockStream) SetUsername(username string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.username = username
}

// Domain returns current mocked stream domain.
func (m *MockStream) Domain() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.domain
}

// SetDomain sets the mocked stream domain value.
func (m *MockStream) SetDomain(domain string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.domain = domain
}

// Resource returns current mocked stream resource.
func (m *MockStream) Resource() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resource
}

// SetResource sets the mocked stream resource value.
func (m *MockStream) SetResource(resource string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resource = resource
}

// JID returns current user JID.
func (m *MockStream) JID() *xml.JID {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.jid
}

// SetJID sets the mocked stream JID value.
func (m *MockStream) SetJID(jid *xml.JID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jid = jid
}

// Priority returns current presence priority.
func (m *MockStream) Priority() int8 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.priority
}

// SetPriority sets mocked stream priority.
func (m *MockStream) SetPriority(priority int8) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.priority = priority
}

// Disconnect disconnects mocked stream.
func (m *MockStream) Disconnect(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.discCh <- err
	m.disconnected = true
}

// IsDisconnected returns whether or not the mocked stream has been disconnected.
func (m *MockStream) IsDisconnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.disconnected
}

// WaitDisconnection waits until the mocked stream disconnects.
func (m *MockStream) WaitDisconnection() error {
	return <-m.discCh
}

// SetSecured sets whether or not the a mocked stream
// has been secured.
func (m *MockStream) SetSecured(secured bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secured = secured
}

// IsSecured returns whether or not the mocked stream
// has been secured.
func (m *MockStream) IsSecured() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.secured
}

// SetAuthenticated sets whether or not the a mocked stream
// has been authenticated.
func (m *MockStream) SetAuthenticated(authenticated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authenticated = authenticated
}

// IsAuthenticated returns whether or not the mocked stream
// has successfully authenticated.
func (m *MockStream) IsAuthenticated() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.authenticated
}

// SetCompressed sets whether or not the a mocked stream
// has been compressed.
func (m *MockStream) SetCompressed(compressed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.compressed = compressed
}

// IsCompressed returns whether or not the mocked stream
// has enabled a compression method.
func (m *MockStream) IsCompressed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.compressed
}

// SetRosterRequested sets whether or not the a mocked stream
// roster has been requested.
func (m *MockStream) SetRosterRequested(rosterRequested bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rosterRequested = rosterRequested
}

// IsRosterRequested returns whether or not user's roster has been requested.
func (m *MockStream) IsRosterRequested() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rosterRequested
}

// PresenceElements returns last available sent presence sub elements.
func (m *MockStream) PresenceElements() []xml.XElement {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.presenceElements
}

// SetPresenceElements sets the mocked stream last received
// presence elements.
func (m *MockStream) SetPresenceElements(presenceElements []xml.XElement) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.presenceElements = presenceElements
}

// SendElement sends the given XML element.
func (m *MockStream) SendElement(element xml.XElement) {
	m.elemCh <- element
}

// FetchElement waits until a new XML element is sent to
// the mocked stream and returns it.
func (m *MockStream) FetchElement() xml.XElement {
	return <-m.elemCh
}
