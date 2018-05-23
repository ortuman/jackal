/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"io"
	"net"
	"testing"
	"time"

	"crypto/tls"

	"github.com/ortuman/jackal/module/offline"
	"github.com/ortuman/jackal/module/xep0077"
	"github.com/ortuman/jackal/module/xep0092"
	"github.com/ortuman/jackal/module/xep0199"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/server/compress"
	"github.com/ortuman/jackal/server/transport"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

type fakeSockReaderWriter struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func newFakeSockReaderWriter() *fakeSockReaderWriter {
	pr, pw := io.Pipe()
	frw := &fakeSockReaderWriter{r: pr, w: pw}
	return frw
}

func (frw *fakeSockReaderWriter) Write(b []byte) (n int, err error) { return frw.w.Write(b) }
func (frw *fakeSockReaderWriter) Read(b []byte) (n int, err error)  { return frw.r.Read(b) }

type fakeSocketConn struct {
	rd      *fakeSockReaderWriter
	wr      *fakeSockReaderWriter
	wrCh    chan []byte
	closeCh chan struct{}
}

func newFakeSocketConn() *fakeSocketConn {
	fc := &fakeSocketConn{
		rd:      newFakeSockReaderWriter(),
		wr:      newFakeSockReaderWriter(),
		wrCh:    make(chan []byte, 16),
		closeCh: make(chan struct{}, 1),
	}
	go fc.loop()
	return fc
}

func (c *fakeSocketConn) Read(b []byte) (n int, err error) { return c.rd.Read(b) }
func (c *fakeSocketConn) Write(b []byte) (n int, err error) {
	wb := make([]byte, len(b))
	copy(wb, b)
	c.wrCh <- wb
	return len(wb), nil
}
func (c *fakeSocketConn) Close() error                       { close(c.closeCh); return nil }
func (c *fakeSocketConn) LocalAddr() net.Addr                { return localAddr }
func (c *fakeSocketConn) RemoteAddr() net.Addr               { return remoteAddr }
func (c *fakeSocketConn) SetDeadline(t time.Time) error      { return nil }
func (c *fakeSocketConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *fakeSocketConn) SetWriteDeadline(t time.Time) error { return nil }

func (c *fakeSocketConn) inboundWrite(b []byte) (n int, err error) {
	return c.rd.Write(b)
}

func (c *fakeSocketConn) parseOutboundElement() xml.XElement {
	var elem xml.XElement
	var err error
	p := xml.NewParser(c.wr, 0)
	for err == nil {
		elem, err = p.ParseElement()
		if elem != nil {
			return elem
		}
	}
	return &xml.Element{}
}

func (c *fakeSocketConn) waitClose() bool {
	select {
	case <-c.closeCh:
		return true
	case <-time.After(time.Second * 5):
		return false // timed out
	}
}

func (c *fakeSocketConn) loop() {
	for {
		select {
		case b := <-c.wrCh:
			c.wr.Write(b)
		case <-c.closeCh:
			return
		}
	}
}

type fakeAddr int

var (
	localAddr  = fakeAddr(1)
	remoteAddr = fakeAddr(2)
)

func (a fakeAddr) Network() string { return "net" }
func (a fakeAddr) String() string  { return "str" }

func TestStream_ConnectTimeout(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	router.Initialize(&router.Config{Domains: []string{"localhost"}})
	defer router.Shutdown()

	stm, _ := tUtilStreamInit()
	time.Sleep(time.Second * 2)
	require.Equal(t, disconnected, stm.getState())
}

func TestStream_Disconnect(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	router.Initialize(&router.Config{Domains: []string{"localhost"}})
	defer router.Shutdown()

	stm, conn := tUtilStreamInit()
	stm.Disconnect(nil)
	require.True(t, conn.waitClose())

	require.Equal(t, disconnected, stm.getState())
}

func TestStream_Features(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	router.Initialize(&router.Config{Domains: []string{"localhost"}})
	defer router.Shutdown()

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)

	elem := conn.parseOutboundElement()
	require.Equal(t, "stream:stream", elem.Name())

	elem = conn.parseOutboundElement()
	require.Equal(t, "stream:features", elem.Name())

	require.Equal(t, connected, stm.getState())
}

func TestStream_TLS(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	router.Initialize(&router.Config{Domains: []string{"localhost"}})
	defer router.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	conn.inboundWrite([]byte(`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`))

	elem := conn.parseOutboundElement()

	require.Equal(t, "proceed", elem.Name())
	require.Equal(t, "urn:ietf:params:xml:ns:xmpp-tls", elem.Namespace())

	require.True(t, stm.IsSecured())
}

func TestStream_Compression(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	router.Initialize(&router.Config{Domains: []string{"localhost"}})
	defer router.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	conn.inboundWrite([]byte(`<compress xmlns="http://jabber.org/protocol/compress">
<method>zlib</method>
</compress>`))

	elem := conn.parseOutboundElement()
	require.Equal(t, "compressed", elem.Name())
	require.Equal(t, "http://jabber.org/protocol/compress", elem.Namespace())

	require.True(t, stm.IsCompressed())
}

func TestStream_StartSession(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	router.Initialize(&router.Config{Domains: []string{"localhost"}})
	defer router.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())
}

func TestStream_SendIQ(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	router.Initialize(&router.Config{Domains: []string{"localhost"}})
	defer router.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())

	// request roster...
	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.GetType)
	iq.AppendElement(xml.NewElementNamespace("query", "jabber:iq:roster"))

	conn.inboundWrite([]byte(iq.String()))

	elem := conn.parseOutboundElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
	require.NotNil(t, elem.Elements().ChildNamespace("query", "jabber:iq:roster"))

	require.True(t, stm.Context().Bool("roster:requested"))
}

func TestStream_SendPresence(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	router.Initialize(&router.Config{Domains: []string{"localhost"}})
	defer router.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())

	conn.inboundWrite([]byte(`
<presence>
<show>away</show>
<status>away!</status>
<priority>5</priority>
<x xmlns="vcard-temp:x:update">
<photo>b7d050434f5441e377dc57f72ac5239e1f493fd0</photo>
</x>
</presence>
	`))
	time.Sleep(time.Millisecond * 100) // wait until processed...

	p := stm.Presence()
	require.NotNil(t, p)
	require.Equal(t, int8(5), p.Priority())
	x := xml.NewElementName("x")
	x.AppendElements(stm.Presence().Elements().All())
	require.NotNil(t, x.Elements().Child("show"))
	require.NotNil(t, x.Elements().Child("status"))
	require.NotNil(t, x.Elements().Child("priority"))
	require.NotNil(t, x.Elements().Child("x"))
}

func TestStream_SendMessage(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	router.Initialize(&router.Config{Domains: []string{"localhost"}})
	defer router.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.parseOutboundElement() // read stream opening...
	_ = conn.parseOutboundElement() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())

	// define a second stream...
	jFrom, _ := xml.NewJID("user", "localhost", "balcony", true)
	jTo, _ := xml.NewJID("ortuman", "localhost", "garden", true)

	stm2 := router.NewMockC2S("abcd7890", jTo)
	router.Instance().RegisterStream(stm2)
	router.Instance().AuthenticateStream(stm2)

	msgID := uuid.New()
	msg := xml.NewMessageType(msgID, xml.ChatType)
	msg.SetFromJID(jFrom)
	msg.SetToJID(jTo)
	body := xml.NewElementName("body")
	body.SetText("Hi buddy!")
	msg.AppendElement(body)

	conn.inboundWrite([]byte(msg.String()))

	// to full jid...
	elem := stm2.FetchElement()
	require.Equal(t, "message", elem.Name())
	require.Equal(t, msgID, elem.ID())

	// to bare jid...
	msg.SetToJID(jTo.ToBareJID())
	conn.inboundWrite([]byte(msg.String()))
	elem = stm2.FetchElement()
	require.Equal(t, "message", elem.Name())
	require.Equal(t, msgID, elem.ID())
}

func tUtilStreamOpen(conn *fakeSocketConn) {
	s := `<?xml version="1.0"?>
	<stream:stream xmlns:stream="http://etherx.jabber.org/streams"
	version="1.0" xmlns="jabber:client" to="localhost" xml:lang="en" xmlns:xml="http://www.w3.org/XML/1998/namespace">
`
	conn.inboundWrite([]byte(s))
}

func tUtilStreamAuthenticate(conn *fakeSocketConn, t *testing.T) {
	conn.inboundWrite([]byte(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="DIGEST-MD5"/>`))

	elem := conn.parseOutboundElement()
	require.Equal(t, "challenge", elem.Name())

	conn.inboundWrite([]byte(`<response xmlns="urn:ietf:params:xml:ns:xmpp-sasl">dXNlcm5hbWU9InVzZXIiLHJlYWxtPSJsb2NhbGhvc3QiLG5vbmNlPSJuY3prcXJFb3Uyait4ek1pcUgxV1lBdHh6dlNCSzFVbHNOejNLQUJsSjd3PSIsY25vbmNlPSJlcHNMSzhFQU8xVWVFTUpLVjdZNXgyYUtqaHN2UXpSMGtIdFM0ZGljdUFzPSIsbmM9MDAwMDAwMDEsZGlnZXN0LXVyaT0ieG1wcC9sb2NhbGhvc3QiLHFvcD1hdXRoLHJlc3BvbnNlPTVmODRmNTk2YWE4ODc0OWY2ZjZkZTYyZjliNjhkN2I2LGNoYXJzZXQ9dXRmLTg=</response>`))

	elem = conn.parseOutboundElement()
	require.Equal(t, "challenge", elem.Name())

	conn.inboundWrite([]byte(`<response xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>`))

	elem = conn.parseOutboundElement()
	require.Equal(t, "success", elem.Name())
}

func tUtilStreamStartSession(conn *fakeSocketConn, t *testing.T) {
	conn.inboundWrite([]byte(`<iq type="set" id="bind_1">
<bind xmlns="urn:ietf:params:xml:ns:xmpp-bind">
<resource>balcony</resource>
</bind>
</iq>`))

	elem := conn.parseOutboundElement()
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().Child("bind"))

	// open session
	conn.inboundWrite([]byte(`<iq type="set" id="aab8a">
<session xmlns="urn:ietf:params:xml:ns:xmpp-session"/>
</iq>`))

	elem = conn.parseOutboundElement()
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, xml.ResultType, elem.Type())

	time.Sleep(time.Millisecond * 100) // wait until stream internal state changes
}

func tUtilStreamInit() (*c2sStream, *fakeSocketConn) {
	conn := newFakeSocketConn()
	tr := transport.NewSocketTransport(conn, 4096)
	stm := newC2SStream("abcd1234", tr, &tls.Config{}, tUtilStreamDefaultConfig())
	router.Instance().RegisterStream(stm)
	return stm, conn
}

func tUtilStreamDefaultConfig() *Config {
	modules := map[string]struct{}{}
	modules["roster"] = struct{}{}
	modules["private"] = struct{}{}
	modules["vcard"] = struct{}{}
	modules["registration"] = struct{}{}
	modules["version"] = struct{}{}
	modules["ping"] = struct{}{}
	modules["offline"] = struct{}{}

	return &Config{
		ID:               "server-id:1234",
		ResourceConflict: Reject,
		Type:             C2SServerType,
		Transport: TransportConfig{
			Type:           transport.Socket,
			ConnectTimeout: 1,
			KeepAlive:      5,
		},
		TLS: TLSConfig{
			PrivKeyFile: "../testdata/cert/test.server.key",
			CertFile:    "../testdata/cert/test.server.crt",
		},
		Compression:     CompressConfig{Level: compress.DefaultCompression},
		SASL:            []string{"plain", "digest_md5", "scram_sha_1", "scram_sha_256"},
		Modules:         modules,
		ModOffline:      offline.Config{QueueSize: 10},
		ModRegistration: xep0077.Config{AllowRegistration: true, AllowChange: true},
		ModVersion:      xep0092.Config{ShowOS: true},
		ModPing:         xep0199.Config{SendInterval: 5, Send: true},
	}
}
