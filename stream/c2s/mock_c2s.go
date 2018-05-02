/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"time"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
)

// MockStream represents a mocked c2s stream.
type MockStream struct {
	id     string
	ctx    *stream.Context
	elemCh chan xml.XElement
	discCh chan error
}

// NewMockStream returns a new mocked stream instance.
func NewMockStream(id string, jid *xml.JID) *MockStream {
	stm := &MockStream{
		id:  id,
		ctx: stream.NewContext(),
	}
	stm.ctx.SetObject(jid, "jid")
	stm.ctx.SetString(jid.Node(), "username")
	stm.ctx.SetString(jid.Domain(), "domain")
	stm.ctx.SetString(jid.Resource(), "resource")
	stm.elemCh = make(chan xml.XElement, 16)
	stm.discCh = make(chan error, 1)
	return stm
}

// ID returns mocked stream identifier.
func (m *MockStream) ID() string {
	return m.id
}

// Context returns mocked stream associated context.
func (m *MockStream) Context() *stream.Context {
	return m.ctx
}

// Username returns current mocked stream username.
func (m *MockStream) Username() string {
	return m.ctx.String("username")
}

// SetUsername sets the mocked stream username value.
func (m *MockStream) SetUsername(username string) {
	m.ctx.SetString(username, "username")
}

// Domain returns current mocked stream domain.
func (m *MockStream) Domain() string {
	return m.ctx.String("domain")
}

// SetDomain sets the mocked stream domain value.
func (m *MockStream) SetDomain(domain string) {
	m.ctx.SetString(domain, "domain")
}

// Resource returns current mocked stream resource.
func (m *MockStream) Resource() string {
	return m.ctx.String("resource")
}

// SetResource sets the mocked stream resource value.
func (m *MockStream) SetResource(resource string) {
	m.ctx.SetString(resource, "resource")
}

// JID returns current user JID.
func (m *MockStream) JID() *xml.JID {
	return m.ctx.Object("jid").(*xml.JID)
}

// SetJID sets the mocked stream JID value.
func (m *MockStream) SetJID(jid *xml.JID) {
	m.ctx.SetObject(jid, "jid")
}

// SetSecured sets whether or not the a mocked stream
// has been secured.
func (m *MockStream) SetSecured(secured bool) {
	m.ctx.SetBool(secured, "secured")
}

// IsSecured returns whether or not the mocked stream
// has been secured.
func (m *MockStream) IsSecured() bool {
	return m.ctx.Bool("secured")
}

// SetAuthenticated sets whether or not the a mocked stream
// has been authenticated.
func (m *MockStream) SetAuthenticated(authenticated bool) {
	m.ctx.SetBool(authenticated, "authenticated")
}

// IsAuthenticated returns whether or not the mocked stream
// has successfully authenticated.
func (m *MockStream) IsAuthenticated() bool {
	return m.ctx.Bool("authenticated")
}

// SetCompressed sets whether or not the a mocked stream
// has been compressed.
func (m *MockStream) SetCompressed(compressed bool) {
	m.ctx.SetBool(compressed, "compressed")
}

// IsCompressed returns whether or not the mocked stream
// has enabled a compression method.
func (m *MockStream) IsCompressed() bool {
	return m.ctx.Bool("compressed")
}

// SetPresence sets the mocked stream last received
// presence element.
func (m *MockStream) SetPresence(presence *xml.Presence) {
	m.ctx.SetObject(presence, "presence")
}

// Presence returns last sent presence element.
func (m *MockStream) Presence() *xml.Presence {
	switch v := m.ctx.Object("presence").(type) {
	case *xml.Presence:
		return v
	}
	return nil
}

// SendElement sends the given XML element.
func (m *MockStream) SendElement(element xml.XElement) {
	m.elemCh <- element
}

// FetchElement waits until a new XML element is sent to
// the mocked stream and returns it.
func (m *MockStream) FetchElement() xml.XElement {
	select {
	case e := <-m.elemCh:
		return e
	case <-time.After(time.Second * 3):
		return &xml.Element{}
	}
}

// Disconnect disconnects mocked stream.
func (m *MockStream) Disconnect(err error) {
	m.discCh <- err
	m.ctx.SetBool(true, "disconnected")
}

// IsDisconnected returns whether or not the mocked stream has been disconnected.
func (m *MockStream) IsDisconnected() bool {
	return m.ctx.Bool("disconnected")
}

// WaitDisconnection waits until the mocked stream disconnects.
func (m *MockStream) WaitDisconnection() error {
	return <-m.discCh
}
