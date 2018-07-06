/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package session

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	stdxml "encoding/xml"
	"io"
	"testing"

	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/transport/compress"
	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type fakeTransport struct {
	typ   transport.TransportType
	rdBuf *bytes.Buffer
	wrBuf *bytes.Buffer
}

func newFakeTransport(typ transport.TransportType) *fakeTransport {
	return &fakeTransport{typ: typ, rdBuf: new(bytes.Buffer), wrBuf: new(bytes.Buffer)}
}

func (t *fakeTransport) Read(p []byte) (n int, err error)                             { return t.rdBuf.Read(p) }
func (t *fakeTransport) Write(p []byte) (n int, err error)                            { return t.wrBuf.Write(p) }
func (t *fakeTransport) Close() error                                                 { return nil }
func (t *fakeTransport) Type() transport.TransportType                                { return t.typ }
func (t *fakeTransport) WriteString(s string) (n int, err error)                      { return t.wrBuf.WriteString(s) }
func (t *fakeTransport) StartTLS(cfg *tls.Config, asClient bool)                      {}
func (t *fakeTransport) EnableCompression(compress.Level)                             {}
func (t *fakeTransport) ChannelBindingBytes(transport.ChannelBindingMechanism) []byte { return nil }
func (t *fakeTransport) PeerCertificates() []*x509.Certificate                        { return nil }

func TestSession_Open(t *testing.T) {
	j, _ := jid.NewWithString("jackal.im", true)

	// test client socket session start
	tr := newFakeTransport(transport.Socket)
	sess := New(uuid.New(), &Config{JID: j, Transport: tr})

	require.NotNil(t, sess.Close())
	_, err1 := sess.Receive()
	require.NotNil(t, err1)

	sess.Open()
	pr := xml.NewParser(tr.wrBuf, xml.SocketStream, 0)
	_, _ = pr.ParseElement() // read xml header
	elem, err := pr.ParseElement()
	require.Nil(t, err)
	require.Equal(t, "stream:stream", elem.Name())
	require.Equal(t, "jabber:client", elem.Namespace())
	require.Equal(t, "http://etherx.jabber.org/streams", elem.Attributes().Get("xmlns:stream"))

	// test server socket session start
	tr.wrBuf.Reset()
	sess = New(uuid.New(), &Config{JID: j, Transport: tr, IsServer: true})
	sess.Open()
	pr = xml.NewParser(tr.wrBuf, xml.SocketStream, 0)
	_, _ = pr.ParseElement() // read xml header
	elem, err = pr.ParseElement()
	require.Nil(t, err)
	require.Equal(t, "jabber:server", elem.Namespace())

	// test websocket session start
	tr = newFakeTransport(transport.WebSocket)
	sess = New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()
	pr = xml.NewParser(tr.wrBuf, xml.WebSocketStream, 0)
	elem, err = pr.ParseElement()
	require.Nil(t, err)
	require.Equal(t, "open", elem.Name())
	require.Equal(t, "urn:ietf:params:xml:ns:xmpp-framing", elem.Attributes().Get("xmlns"))

	// test unsupported transport type
	tr = newFakeTransport(transport.TransportType(9999))
	sess = New(uuid.New(), &Config{JID: j, Transport: tr})
	require.Nil(t, sess.Open())

	// open twice
	require.NotNil(t, sess.Open())
}

func TestSession_Close(t *testing.T) {
	j, _ := jid.NewWithString("jackal.im", true)

	tr := newFakeTransport(transport.Socket)
	sess := New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()
	tr.wrBuf.Reset()

	sess.Close()
	require.Equal(t, "</stream:stream>", tr.wrBuf.String())

	tr = newFakeTransport(transport.WebSocket)
	sess = New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()
	tr.wrBuf.Reset()

	sess.Close()
	require.Equal(t, `<close xmlns="urn:ietf:params:xml:ns:xmpp-framing" />`, tr.wrBuf.String())
}

func TestSession_Send(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/res", true)
	tr := newFakeTransport(transport.Socket)
	sess := New(uuid.New(), &Config{JID: j, Transport: tr})
	elem := xml.NewElementNamespace("open", "urn:ietf:params:xml:ns:xmpp-framing")
	sess.Open()
	tr.wrBuf.Reset()

	sess.Send(elem)
	require.Equal(t, elem.String(), tr.wrBuf.String())
}

func TestSession_Receive(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/res", true)
	tr := newFakeTransport(transport.WebSocket)
	sess := New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()
	_, err := sess.Receive()
	require.Equal(t, &Error{}, err)

	tr = newFakeTransport(transport.WebSocket)
	sess = New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()
	open := xml.NewElementNamespace("open", "")
	open.ToXML(tr.rdBuf, true)

	_, err = sess.Receive()
	require.Equal(t, &Error{UnderlyingErr: streamerror.ErrInvalidNamespace}, err)

	tr = newFakeTransport(transport.WebSocket)
	sess = New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()
	open.SetNamespace("urn:ietf:params:xml:ns:xmpp-framing")
	open.SetVersion("1.0")
	open.ToXML(tr.rdBuf, true)

	iq := xml.NewIQType(uuid.New(), xml.ResultType)
	iq.ToXML(tr.rdBuf, true)

	_, err = sess.Receive()   // read open stream element...
	st, err := sess.Receive() // read IQ...
	require.Nil(t, err)
	require.Equal(t, "iq", st.Name())

	tr = newFakeTransport(transport.WebSocket)
	sess = New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()
	open.ToXML(tr.rdBuf, true)

	// bad stanza
	xml.NewElementName("iq").ToXML(tr.rdBuf, true)
	_, err = sess.Receive() // read open stream element...
	_, err = sess.Receive()
	require.NotNil(t, err)
	require.Equal(t, xml.ErrBadRequest, err.UnderlyingErr)
}

func TestSession_IsValidNamespace(t *testing.T) {
	iqClient := xml.NewElementNamespace("iq", "jabber:client")
	iqServer := xml.NewElementNamespace("iq", "jabber:server")

	j, _ := jid.NewWithString("jackal.im", true)

	tr := newFakeTransport(transport.Socket)
	sess := New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()
	require.Nil(t, sess.validateNamespace(iqClient))
	require.Equal(t, &Error{UnderlyingErr: streamerror.ErrInvalidNamespace}, sess.validateNamespace(iqServer))

	tr = newFakeTransport(transport.Socket)
	sess = New(uuid.New(), &Config{JID: j, Transport: tr, IsServer: true})
	sess.Open()
	require.Equal(t, &Error{UnderlyingErr: streamerror.ErrInvalidNamespace}, sess.validateNamespace(iqClient))
	require.Nil(t, sess.validateNamespace(iqServer))
}

func TestSession_IsValidFrom(t *testing.T) {
	j1, _ := jid.NewWithString("jackal.im", true)                  // server domain
	j2, _ := jid.NewWithString("ortuman@jackal.im/resource", true) // full jid with user

	tr := newFakeTransport(transport.Socket)
	sess := New(uuid.New(), &Config{JID: j2, Transport: tr})
	sess.Open()
	sess.SetJID(j1)
	require.False(t, sess.isValidFrom("romeo@jackal.im"))

	sess.SetJID(j2)
	require.True(t, sess.isValidFrom("ortuman@jackal.im/resource"))
}

func TestSession_ValidateStream(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	defer func() {
		host.Shutdown()
	}()

	j, _ := jid.NewWithString("jackal.im", true) // server domain

	elem1 := xml.NewElementNamespace("stream:stream", "")
	elem2 := xml.NewElementNamespace("stream:stream", "jabber:client")
	elem4 := xml.NewElementNamespace("open", "")
	elem5 := xml.NewElementNamespace("open", "urn:ietf:params:xml:ns:xmpp-framing")

	// try socket
	tr := newFakeTransport(transport.Socket)
	sess := New(uuid.New(), &Config{JID: j, Transport: tr})
	err := sess.validateStreamElement(elem1)
	sess.Open()
	require.NotNil(t, err)
	require.Equal(t, streamerror.ErrInvalidNamespace, err.UnderlyingErr)

	err = sess.validateStreamElement(elem2)
	require.NotNil(t, err)
	require.Equal(t, streamerror.ErrInvalidNamespace, err.UnderlyingErr)

	err = sess.validateStreamElement(elem4)
	require.NotNil(t, err)
	require.Equal(t, streamerror.ErrUnsupportedStanzaType, err.UnderlyingErr)

	elem2.SetAttribute("xmlns:stream", "http://etherx.jabber.org/streams")
	err = sess.validateStreamElement(elem2)
	require.NotNil(t, err)

	elem2.SetVersion("1.0")
	elem2.SetTo("example.org")

	err = sess.validateStreamElement(elem2)
	require.NotNil(t, err)
	require.Equal(t, streamerror.ErrHostUnknown, err.UnderlyingErr)

	elem2.SetTo("jackal.im")
	require.Nil(t, sess.validateStreamElement(elem2))

	// try websocket
	tr = newFakeTransport(transport.WebSocket)
	sess = New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()
	err = sess.validateStreamElement(elem4)
	require.NotNil(t, err)
	require.Equal(t, streamerror.ErrInvalidNamespace, err.UnderlyingErr)

	err = sess.validateStreamElement(elem1)
	require.NotNil(t, err)
	require.Equal(t, streamerror.ErrUnsupportedStanzaType, err.UnderlyingErr)

	err = sess.validateStreamElement(elem5)
	require.NotNil(t, err)
	require.Equal(t, streamerror.ErrUnsupportedVersion, err.UnderlyingErr)

	elem5.SetVersion("1.0")
	elem5.SetTo("example.org")

	err = sess.validateStreamElement(elem5)
	require.NotNil(t, err)
	require.Equal(t, streamerror.ErrHostUnknown, err.UnderlyingErr)

	elem5.SetTo("jackal.im")
	require.Nil(t, sess.validateStreamElement(elem5))
}

func TestSession_ExtractAddresses(t *testing.T) {
	j1, _ := jid.NewWithString("jackal.im", true)
	j2, _ := jid.NewWithString("ortuman@jackal.im/res", true)

	iq := xml.NewElementNamespace("iq", "jabber:client")
	iq.SetFrom("ortuman@jackal.im/res")
	iq.SetTo("romeo@example.org")

	tr := newFakeTransport(transport.Socket)
	sess := New(uuid.New(), &Config{JID: j1, Transport: tr})
	sess.Open()
	from, to, err := sess.extractAddresses(iq)
	require.Nil(t, err)
	require.Equal(t, "jackal.im", from.String())
	require.Equal(t, "romeo@example.org", to.String())

	sess.SetJID(j2)

	iq.SetFrom("romeo@example.org")
	iq.SetTo("")
	_, _, err = sess.extractAddresses(iq)
	require.Equal(t, streamerror.ErrInvalidFrom, err.UnderlyingErr)

	iq.SetFrom("ortuman@jackal.im/res")
	iq.SetTo("")
	from, to, err = sess.extractAddresses(iq)
	require.Nil(t, err)
	require.Equal(t, "ortuman@jackal.im/res", from.String())
	require.Equal(t, "ortuman@jackal.im", to.String())

	iq.SetTo("ortuman@" + string([]byte{255, 255, 255}) + "/res")
	_, _, err = sess.extractAddresses(iq)
	require.NotNil(t, err)
	require.Equal(t, iq, err.Element)
	require.Equal(t, xml.ErrJidMalformed, err.UnderlyingErr)
}

func TestSession_BuildStanza(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/res", true)
	tr := newFakeTransport(transport.Socket)
	sess := New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()

	elem := xml.NewElementNamespace("n", "ns")
	_, err := sess.buildStanza(elem)
	require.NotNil(t, err)
	require.Equal(t, streamerror.ErrInvalidNamespace, err.UnderlyingErr)

	elem.SetNamespace("")
	_, err = sess.buildStanza(elem)
	require.NotNil(t, err)
	require.Equal(t, streamerror.ErrUnsupportedStanzaType, err.UnderlyingErr)

	elem.SetName("iq")
	elem.SetTo("ortuman@" + string([]byte{255, 255, 255}) + "/res")
	_, err = sess.buildStanza(elem)
	require.NotNil(t, err)
	require.Equal(t, xml.ErrJidMalformed, err.UnderlyingErr)

	elem.SetTo("ortuman@jackal.im/res")
	_, err = sess.buildStanza(elem)
	require.NotNil(t, err)
	require.Equal(t, xml.ErrBadRequest, err.UnderlyingErr)

	elem.SetID(uuid.New())
	elem.SetType("result")
	_, err = sess.buildStanza(elem)
	require.Nil(t, err)

	elem.SetName("presence")
	_, err = sess.buildStanza(elem)
	require.NotNil(t, err)
	require.Equal(t, xml.ErrBadRequest, err.UnderlyingErr)

	elem.SetType("unavailable")
	_, err = sess.buildStanza(elem)
	require.Nil(t, err)

	elem.SetName("message")
	_, err = sess.buildStanza(elem)
	require.NotNil(t, err)
	require.Equal(t, xml.ErrBadRequest, err.UnderlyingErr)

	elem.SetType("normal")
	_, err = sess.buildStanza(elem)
	require.Nil(t, err)
}

func TestSession_MapErrorq(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/res", true)
	tr := newFakeTransport(transport.Socket)
	sess := New(uuid.New(), &Config{JID: j, Transport: tr})
	sess.Open()

	require.Equal(t, &Error{}, sess.mapErrorToSessionError(nil))
	require.Equal(t, &Error{}, sess.mapErrorToSessionError(io.EOF))
	require.Equal(t, &Error{}, sess.mapErrorToSessionError(io.ErrUnexpectedEOF))
	require.Equal(t, &Error{}, sess.mapErrorToSessionError(xml.ErrStreamClosedByPeer))

	require.Equal(t, &Error{UnderlyingErr: streamerror.ErrPolicyViolation}, sess.mapErrorToSessionError(xml.ErrTooLargeStanza))
	require.Equal(t, &Error{UnderlyingErr: streamerror.ErrInvalidXML}, sess.mapErrorToSessionError(&stdxml.SyntaxError{}))

	er := errors.New("err")
	require.Equal(t, &Error{UnderlyingErr: er}, sess.mapErrorToSessionError(er))
}
