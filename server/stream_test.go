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
	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	stm, conn := tUtilStreamInit()
	conn.WaitCloseWithTimeout(time.Second * 2)
	require.Equal(t, disconnected, stm.getState())
}

func TestStream_Disconnect(t *testing.T) {
	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	stm, conn := tUtilStreamInit()
	stm.Disconnect(nil)
	conn.WaitClose()

	require.Equal(t, disconnected, stm.getState())
}

func TestStream_Features(t *testing.T) {
	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	time.Sleep(time.Millisecond * 50) // wait for write...

	elems := conn.ReadElements()
	require.Equal(t, 2, len(elems))
	require.Equal(t, "stream:stream", elems[0].Name())
	require.Equal(t, "stream:features", elems[1].Name())

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
	elems := conn.ReadElements()

	conn.SendBytes([]byte(`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`))
	time.Sleep(time.Millisecond * 50)

	elems = conn.ReadElements()
	require.Equal(t, 1, len(elems))
	require.Equal(t, "proceed", elems[0].Name())
	require.Equal(t, "urn:ietf:params:xml:ns:xmpp-tls", elems[0].Namespace())

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
	_ = conn.ReadElements()

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.ReadElements()

	conn.SendBytes([]byte(`<compress xmlns="http://jabber.org/protocol/compress">
<method>zlib</method>
</compress>`))
	time.Sleep(time.Millisecond * 50)

	elems := conn.ReadElements()
	require.Equal(t, 1, len(elems))
	require.Equal(t, "compressed", elems[0].Name())
	require.Equal(t, "http://jabber.org/protocol/compress", elems[0].Namespace())

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
	_ = conn.ReadElements()

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.ReadElements()

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
	_ = conn.ReadElements()

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.ReadElements()

	tUtilStreamStartSession(conn, t)
	require.Equal(t, sessionStarted, stm.getState())

	// request roster...
	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.GetType)
	iq.AppendElement(xml.NewElementNamespace("query", "jabber:iq:roster"))

	conn.SendBytes([]byte(iq.String()))
	time.Sleep(time.Millisecond * 50)

	elems := conn.ReadElements()
	require.Equal(t, 1, len(elems))
	require.Equal(t, "iq", elems[0].Name())
	require.Equal(t, iqID, elems[0].ID())
	require.NotNil(t, elems[0].FindElementsNamespace("query", "jabber:iq:roster"))

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
	_ = conn.ReadElements()

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.ReadElements()

	tUtilStreamStartSession(conn, t)
	require.Equal(t, sessionStarted, stm.getState())

	conn.SendBytes([]byte(`
<presence>
<show>away</show>
<status>away!</status>
<priority>5</priority>
<x xmlns="vcard-temp:x:update">
<photo>b7d050434f5441e377dc57f72ac5239e1f493fd0</photo>
</x>
</presence>
	`))
	time.Sleep(time.Millisecond * 50)

	require.Equal(t, int8(5), stm.Priority())
	x := xml.NewElementName("x")
	x.AppendElements(stm.PresenceElements())
	require.NotNil(t, x.FindElement("show"))
	require.NotNil(t, x.FindElement("status"))
	require.NotNil(t, x.FindElement("priority"))
	require.NotNil(t, x.FindElement("x"))
}

func TestStream_SendMessage(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"localhost"}})
	defer c2s.Shutdown()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit()
	tUtilStreamOpen(conn)
	_ = conn.ReadElements()

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.ReadElements()

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

	conn.SendBytes([]byte(msg.String()))

	// to full jid...
	elem := stm2.FetchElement()
	require.Equal(t, "message", elem.Name())
	require.Equal(t, msgID, elem.ID())

	// to bare jid...
	msg.SetToJID(jTo.ToBareJID())
	conn.SendBytes([]byte(msg.String()))
	elem = stm2.FetchElement()
	require.Equal(t, "message", elem.Name())
	require.Equal(t, msgID, elem.ID())
}

func tUtilStreamOpen(conn *transport.MockConn) {
	s := `<?xml version="1.0"?>
	<stream:stream xmlns:stream="http://etherx.jabber.org/streams" 
	version="1.0" xmlns="jabber:client" to="localhost" xml:lang="en" xmlns:xml="http://www.w3.org/XML/1998/namespace">
`
	conn.SendBytes([]byte(s))
	time.Sleep(time.Millisecond * 50)
}

func tUtilStreamAuthenticate(conn *transport.MockConn, t *testing.T) {
	conn.SendBytes([]byte(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="DIGEST-MD5"/>`))
	time.Sleep(time.Millisecond * 50) // wait for write...

	elems := conn.ReadElements()
	require.Equal(t, 1, len(elems))
	require.Equal(t, "challenge", elems[0].Name())

	conn.SendBytes([]byte(`<response xmlns="urn:ietf:params:xml:ns:xmpp-sasl">dXNlcm5hbWU9InVzZXIiLHJlYWxtPSJsb2NhbGhvc3QiLG5vbmNlPSJuY3prcXJFb3Uyait4ek1pcUgxV1lBdHh6dlNCSzFVbHNOejNLQUJsSjd3PSIsY25vbmNlPSJlcHNMSzhFQU8xVWVFTUpLVjdZNXgyYUtqaHN2UXpSMGtIdFM0ZGljdUFzPSIsbmM9MDAwMDAwMDEsZGlnZXN0LXVyaT0ieG1wcC9sb2NhbGhvc3QiLHFvcD1hdXRoLHJlc3BvbnNlPTVmODRmNTk2YWE4ODc0OWY2ZjZkZTYyZjliNjhkN2I2LGNoYXJzZXQ9dXRmLTg=</response>`))
	time.Sleep(time.Millisecond * 50) // wait for write...

	elems = conn.ReadElements()
	require.Equal(t, 1, len(elems))
	require.Equal(t, "challenge", elems[0].Name())

	conn.SendBytes([]byte(`<response xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>`))
	time.Sleep(time.Millisecond * 50)

	elems = conn.ReadElements()
	require.Equal(t, 1, len(elems))
	require.Equal(t, "success", elems[0].Name())
}

func tUtilStreamStartSession(conn *transport.MockConn, t *testing.T) {
	conn.SendBytes([]byte(`<iq type="set" id="bind_1">
<bind xmlns="urn:ietf:params:xml:ns:xmpp-bind">
<resource>balcony</resource>
</bind>
</iq>`))
	time.Sleep(time.Millisecond * 50)

	elems := conn.ReadElements()
	require.Equal(t, 1, len(elems))
	require.Equal(t, "iq", elems[0].Name())
	require.NotNil(t, elems[0].FindElement("bind"))

	// open session
	conn.SendBytes([]byte(`<iq type="set" id="aab8a">
<session xmlns="urn:ietf:params:xml:ns:xmpp-session"/>
</iq>`))
	time.Sleep(time.Millisecond * 50)

	elems = conn.ReadElements()
	require.Equal(t, 1, len(elems))
	require.Equal(t, "iq", elems[0].Name())
	require.NotNil(t, xml.ResultType, elems[0].Type())
}

func tUtilStreamInit() (*serverStream, *transport.MockConn) {
	conn := transport.NewMockConn()
	stm := newSocketStream("abcd1234", conn, tUtilStreamDefaultConfig())
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
			Type:           config.Socket,
			ConnectTimeout: 1,
			KeepAlive:      5,
		},
		TLS: config.TLS{
			PrivKeyFile: "../cert/key.pem",
			CertFile:    "../cert/cert.pem",
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
