/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"testing"
	"time"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/server/transport"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestStream_ConnectTimeout(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	stm, conn := tUtilStreamInit()
	conn.WaitCloseWithTimeout(time.Second * 2)
	require.Equal(t, disconnected, stm.getState())
}

func TestStream_Disconnect(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	stm, conn := tUtilStreamInit()
	stm.Disconnect(nil)
	conn.WaitClose()

	require.Equal(t, disconnected, stm.getState())
}

func TestStream_Features(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)

	elem := conn.ClientReadElement()
	require.Equal(t, "stream:stream", elem.Name())

	elem = conn.ClientReadElement()
	require.Equal(t, "stream:features", elem.Name())

	require.Equal(t, connected, stm.getState())
}

func TestStream_TLS(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	conn.ClientWriteBytes([]byte(`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`))

	elem := conn.ClientReadElement()

	require.Equal(t, "proceed", elem.Name())
	require.Equal(t, "urn:ietf:params:xml:ns:xmpp-tls", elem.Namespace())

	require.True(t, stm.IsSecured())
}

func TestStream_Compression(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	conn.ClientWriteBytes([]byte(`<compress xmlns="http://jabber.org/protocol/compress">
<method>zlib</method>
</compress>`))

	elem := conn.ClientReadElement()
	require.Equal(t, "compressed", elem.Name())
	require.Equal(t, "http://jabber.org/protocol/compress", elem.Namespace())

	require.True(t, stm.IsCompressed())
}

func TestStream_StartSession(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())
}

func TestStream_SendIQ(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())

	// request roster...
	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.GetType)
	iq.AppendElement(xml.NewElementNamespace("query", "jabber:iq:roster"))

	conn.ClientWriteBytes([]byte(iq.String()))

	elem := conn.ClientReadElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
	require.NotNil(t, elem.Elements().ChildNamespace("query", "jabber:iq:roster"))

	require.True(t, stm.IsRosterRequested())
}

func TestStream_SendPresence(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())

	conn.ClientWriteBytes([]byte(`
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

	require.Equal(t, int8(5), stm.Priority())
	x := xml.NewElementName("x")
	x.AppendElements(stm.Presence().Elements().All())
	require.NotNil(t, x.Elements().Child("show"))
	require.NotNil(t, x.Elements().Child("status"))
	require.NotNil(t, x.Elements().Child("priority"))
	require.NotNil(t, x.Elements().Child("x"))
}

func TestStream_SendMessage(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.ClientReadElement() // read stream opening...
	_ = conn.ClientReadElement() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())

	// define a second stream...
	jFrom, _ := xml.NewJID("user", "localhost", "balcony", true)
	jTo, _ := xml.NewJID("ortuman", "localhost", "garden", true)

	stm2 := c2s.NewMockStream("abcd7890", jTo)
	c2s.Instance().RegisterStream(stm2)
	c2s.Instance().AuthenticateStream(stm2)

	msgID := uuid.New()
	msg := xml.NewMessageType(msgID, xml.ChatType)
	msg.SetFromJID(jFrom)
	msg.SetToJID(jTo)
	body := xml.NewElementName("body")
	body.SetText("Hi buddy!")
	msg.AppendElement(body)

	conn.ClientWriteBytes([]byte(msg.String()))

	// to full jid...
	elem := stm2.FetchElement()
	require.Equal(t, "message", elem.Name())
	require.Equal(t, msgID, elem.ID())

	// to bare jid...
	msg.SetToJID(jTo.ToBareJID())
	conn.ClientWriteBytes([]byte(msg.String()))
	elem = stm2.FetchElement()
	require.Equal(t, "message", elem.Name())
	require.Equal(t, msgID, elem.ID())
}

func tUtilStreamOpen(conn *transport.MockConn) {
	s := `<?xml version="1.0"?>
	<stream:stream xmlns:stream="http://etherx.jabber.org/streams"
	version="1.0" xmlns="jabber:client" to="localhost" xml:lang="en" xmlns:xml="http://www.w3.org/XML/1998/namespace">
`
	conn.ClientWriteBytes([]byte(s))
}

func tUtilStreamAuthenticate(conn *transport.MockConn, t *testing.T) {
	conn.ClientWriteBytes([]byte(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="DIGEST-MD5"/>`))

	elem := conn.ClientReadElement()
	require.Equal(t, "challenge", elem.Name())

	conn.ClientWriteBytes([]byte(`<response xmlns="urn:ietf:params:xml:ns:xmpp-sasl">dXNlcm5hbWU9InVzZXIiLHJlYWxtPSJsb2NhbGhvc3QiLG5vbmNlPSJuY3prcXJFb3Uyait4ek1pcUgxV1lBdHh6dlNCSzFVbHNOejNLQUJsSjd3PSIsY25vbmNlPSJlcHNMSzhFQU8xVWVFTUpLVjdZNXgyYUtqaHN2UXpSMGtIdFM0ZGljdUFzPSIsbmM9MDAwMDAwMDEsZGlnZXN0LXVyaT0ieG1wcC9sb2NhbGhvc3QiLHFvcD1hdXRoLHJlc3BvbnNlPTVmODRmNTk2YWE4ODc0OWY2ZjZkZTYyZjliNjhkN2I2LGNoYXJzZXQ9dXRmLTg=</response>`))

	elem = conn.ClientReadElement()
	require.Equal(t, "challenge", elem.Name())

	conn.ClientWriteBytes([]byte(`<response xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>`))

	elem = conn.ClientReadElement()
	require.Equal(t, "success", elem.Name())
}

func tUtilStreamStartSession(conn *transport.MockConn, t *testing.T) {
	conn.ClientWriteBytes([]byte(`<iq type="set" id="bind_1">
<bind xmlns="urn:ietf:params:xml:ns:xmpp-bind">
<resource>balcony</resource>
</bind>
</iq>`))

	elem := conn.ClientReadElement()
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().Child("bind"))

	// open session
	conn.ClientWriteBytes([]byte(`<iq type="set" id="aab8a">
<session xmlns="urn:ietf:params:xml:ns:xmpp-session"/>
</iq>`))

	elem = conn.ClientReadElement()
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, xml.ResultType, elem.Type())

	time.Sleep(time.Millisecond * 100) // wait until stream internal state changes
}

func tUtilStreamInit() (*c2sStream, *transport.MockConn) {
	conn := transport.NewMockConn()
	tr := transport.NewSocketTransport(conn, 4096, 4096)
	stm := newC2SStream("abcd1234", tr, tUtilStreamDefaultConfig())
	c2s.Instance().RegisterStream(stm)
	return stm, conn
}

func tUtilStreamDefaultConfig() *config.Server {
	modules := map[string]struct{}{}
	modules["roster"] = struct{}{}
	modules["private"] = struct{}{}
	modules["vcard"] = struct{}{}
	modules["registration"] = struct{}{}
	modules["version"] = struct{}{}
	modules["ping"] = struct{}{}
	modules["offline"] = struct{}{}

	return &config.Server{
		ID:               "server-id:1234",
		ResourceConflict: config.Reject,
		Type:             config.C2SServerType,
		Transport: config.Transport{
			Type:           config.SocketTransportType,
			ConnectTimeout: 1,
			KeepAlive:      5,
		},
		TLS: config.TLS{
			PrivKeyFile: "../testdata/cert/test.server.key",
			CertFile:    "../testdata/cert/test.server.crt",
		},
		Compression:     config.Compression{Level: config.DefaultCompression},
		SASL:            []string{"plain", "digest_md5", "scram_sha_1", "scram_sha_256"},
		Modules:         modules,
		ModOffline:      config.ModOffline{QueueSize: 10},
		ModRegistration: config.ModRegistration{AllowRegistration: true, AllowChange: true},
		ModVersion:      config.ModVersion{ShowOS: true},
		ModPing:         config.ModPing{SendInterval: 5, Send: true},
	}
}
