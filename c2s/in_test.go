/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"testing"
	"time"

	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/transport/compress"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestStream_ConnectTimeout(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	stm, _ := tUtilStreamInit(r)
	time.Sleep(time.Millisecond * 1500)
	require.Equal(t, disconnected, stm.getState())
}

func TestStream_Disconnect(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	stm, conn := tUtilStreamInit(r)
	stm.Disconnect(nil)
	require.True(t, conn.waitClose())

	require.Equal(t, disconnected, stm.getState())
}

func TestStream_Features(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	// unsecured features
	stm, conn := tUtilStreamInit(r)
	tUtilStreamOpen(conn)

	elem := conn.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())

	elem = conn.outboundRead()
	require.Equal(t, "stream:features", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("starttls", tlsNamespace))

	require.Equal(t, connected, stm.getState())

	// secured features
	stm2, conn2 := tUtilStreamInit(r)
	stm2.setSecured(true)

	tUtilStreamOpen(conn2)

	elem = conn2.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())

	elem = conn2.outboundRead()
	require.Equal(t, "stream:features", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("mechanisms", saslNamespace))
}

func TestStream_TLS(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	storage.InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r)
	tUtilStreamOpen(conn)

	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	conn.inboundWrite([]byte(`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`))

	elem := conn.outboundRead()

	require.Equal(t, "proceed", elem.Name())
	require.Equal(t, "urn:ietf:params:xml:ns:xmpp-tls", elem.Namespace())

	require.True(t, stm.IsSecured())
}

func TestStream_FailAuthenticate(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	storage.InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	_, conn := tUtilStreamInit(r)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	// wrong mechanism
	conn.inboundWrite([]byte(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="FOO"/>`))

	elem := conn.outboundRead()
	require.Equal(t, "failure", elem.Name())

	conn.inboundWrite([]byte(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="DIGEST-MD5"/>`))

	elem = conn.outboundRead()
	require.Equal(t, "challenge", elem.Name())

	conn.inboundWrite([]byte(`<response xmlns="urn:ietf:params:xml:ns:xmpp-sasl">dXNlcm5hbWU9Im9ydHVtYW4iLHJlYWxtPSJsb2NhbGhvc3QiLG5vbmNlPSJuY3prcXJFb3Uyait4ek1pcUgxV1lBdHh6dlNCSzFVbHNOejNLQUJsSjd3PSIsY25vbmNlPSJlcHNMSzhFQU8xVWVFTUpLVjdZNXgyYUtqaHN2UXpSMGtIdFM0ZGljdUFzPSIsbmM9MDAwMDAwMDEsZGlnZXN0LXVyaT0ieG1wcC9sb2NhbGhvc3QiLHFvcD1hdXRoLHJlc3BvbnNlPTVmODRmNTk2YWE4ODc0OWY2ZjZkZTYyZjliNjhkN2I2LGNoYXJzZXQ9dXRmLTg=</response>`))

	elem = conn.outboundRead()
	require.Equal(t, "failure", elem.Name())

	// non-SASL
	conn.inboundWrite([]byte(`<iq type='set' id='auth2'><query xmlns='jabber:iq:auth'>
<username>bill</username>
<password>Calli0pe</password>
</query>
</iq>`))

	elem = conn.outboundRead()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ErrorType, elem.Type())
	require.NotNil(t, elem.Elements().Child("error"))
}

func TestStream_Compression(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	storage.InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	// no method...
	conn.inboundWrite([]byte(`<compress xmlns="http://jabber.org/protocol/compress"/>`))
	elem := conn.outboundRead()
	require.Equal(t, "failure", elem.Name())
	require.NotNil(t, elem.Elements().Child("setup-failed"))

	// invalid method...
	conn.inboundWrite([]byte(`<compress xmlns="http://jabber.org/protocol/compress">
<method>7z</method>
</compress>`))
	elem = conn.outboundRead()
	require.Equal(t, "failure", elem.Name())
	require.NotNil(t, elem.Elements().Child("unsupported-method"))

	// valid method...
	conn.inboundWrite([]byte(`<compress xmlns="http://jabber.org/protocol/compress">
<method>zlib</method>
</compress>`))

	elem = conn.outboundRead()
	require.Equal(t, "compressed", elem.Name())
	require.Equal(t, "http://jabber.org/protocol/compress", elem.Namespace())

	require.True(t, stm.isCompressed())
}

func TestStream_StartSession(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	storage.InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())
}

func TestStream_SendIQ(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	storage.InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())

	// request roster...
	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.AppendElement(xmpp.NewElementNamespace("query", "jabber:iq:roster"))

	conn.inboundWrite([]byte(iq.String()))

	elem := conn.outboundRead()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
	require.NotNil(t, elem.Elements().ChildNamespace("query", "jabber:iq:roster"))

	require.True(t, stm.GetBool("roster:requested"))
}

func TestStream_SendPresence(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	storage.InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

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
	x := xmpp.NewElementName("x")
	x.AppendElements(stm.Presence().Elements().All())
	require.NotNil(t, x.Elements().Child("show"))
	require.NotNil(t, x.Elements().Child("status"))
	require.NotNil(t, x.Elements().Child("priority"))
	require.NotNil(t, x.Elements().Child("x"))
}

func TestStream_SendMessage(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	storage.InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())

	// define a second stream...
	jFrom, _ := jid.New("user", "localhost", "balcony", true)
	jTo, _ := jid.New("ortuman", "localhost", "garden", true)

	stm2 := stream.NewMockC2S("abcd7890", jTo)
	r.Bind(stm2)

	msgID := uuid.New()
	msg := xmpp.NewMessageType(msgID, xmpp.ChatType)
	msg.SetFromJID(jFrom)
	msg.SetToJID(jTo)
	body := xmpp.NewElementName("body")
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

func TestStream_SendToBlockedJID(t *testing.T) {
	r, _, shutdown := setupTest("localhost")
	defer shutdown()

	storage.InsertOrUpdateUser(&model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamStartSession(conn, t)

	require.Equal(t, sessionStarted, stm.getState())

	storage.InsertBlockListItems([]model.BlockListItem{{
		Username: "user",
		JID:      "hamlet@localhost",
	}})

	// send presence to a blocked JID...
	conn.inboundWrite([]byte(`<presence to="hamlet@localhost"/>`))

	elem := conn.outboundRead()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xmpp.ErrorType, elem.Type())
	require.NotNil(t, elem.Elements().Child("error"))
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

	elem := conn.outboundRead()
	require.Equal(t, "challenge", elem.Name())

	conn.inboundWrite([]byte(`<response xmlns="urn:ietf:params:xml:ns:xmpp-sasl">dXNlcm5hbWU9InVzZXIiLHJlYWxtPSJsb2NhbGhvc3QiLG5vbmNlPSJuY3prcXJFb3Uyait4ek1pcUgxV1lBdHh6dlNCSzFVbHNOejNLQUJsSjd3PSIsY25vbmNlPSJlcHNMSzhFQU8xVWVFTUpLVjdZNXgyYUtqaHN2UXpSMGtIdFM0ZGljdUFzPSIsbmM9MDAwMDAwMDEsZGlnZXN0LXVyaT0ieG1wcC9sb2NhbGhvc3QiLHFvcD1hdXRoLHJlc3BvbnNlPTVmODRmNTk2YWE4ODc0OWY2ZjZkZTYyZjliNjhkN2I2LGNoYXJzZXQ9dXRmLTg=</response>`))

	elem = conn.outboundRead()
	require.Equal(t, "challenge", elem.Name())

	conn.inboundWrite([]byte(`<response xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>`))

	elem = conn.outboundRead()
	require.Equal(t, "success", elem.Name())
}

func tUtilStreamStartSession(conn *fakeSocketConn, t *testing.T) {
	conn.inboundWrite([]byte(`<iq type="set" id="bind_1">
<bind xmlns="urn:ietf:params:xml:ns:xmpp-bind">
<resource>balcony</resource>
</bind>
</iq>`))

	elem := conn.outboundRead()
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().Child("bind"))

	// open session
	conn.inboundWrite([]byte(`<iq type="set" id="aab8a">
<session xmlns="urn:ietf:params:xml:ns:xmpp-session"/>
</iq>`))

	elem = conn.outboundRead()
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, xmpp.ResultType, elem.Type())

	time.Sleep(time.Millisecond * 100) // wait until stream internal state changes
}

func tUtilStreamInit(r *router.Router) (*inStream, *fakeSocketConn) {
	conn := newFakeSocketConn()
	tr := transport.NewSocketTransport(conn, 4096)
	stm := newStream("abcd1234", tUtilInStreamDefaultConfig(tr), tUtilInitModules(r), &component.Components{}, r)
	return stm.(*inStream), conn
}

func tUtilInStreamDefaultConfig(tr transport.Transport) *streamConfig {
	return &streamConfig{
		connectTimeout:   time.Second,
		transport:        tr,
		maxStanzaSize:    8192,
		resourceConflict: Reject,
		compression:      CompressConfig{Level: compress.DefaultCompression},
		sasl:             []string{"plain", "digest_md5", "scram_sha_1", "scram_sha_256"},
	}
}

func tUtilInitModules(r *router.Router) *module.Modules {
	modules := map[string]struct{}{}
	modules["roster"] = struct{}{}
	modules["blocking_command"] = struct{}{}

	return module.New(&module.Config{Enabled: modules}, r)
}
