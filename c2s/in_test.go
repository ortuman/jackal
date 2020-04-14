/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/transport/compress"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestStream_ConnectTimeout(t *testing.T) {
	r, userRep, blockListRep := setupTest("localhost")

	stm, _ := tUtilStreamInit(r, userRep, blockListRep)
	time.Sleep(time.Millisecond * 1500)
	require.Equal(t, disconnected, stm.getState())
}

func TestStream_Disconnect(t *testing.T) {
	r, userRep, blockListRep := setupTest("localhost")

	stm, conn := tUtilStreamInit(r, userRep, blockListRep)
	stm.Disconnect(context.Background(), nil)
	require.True(t, conn.waitClose())

	require.Equal(t, disconnected, stm.getState())
}

func TestStream_Features(t *testing.T) {
	r, userRep, blockListRep := setupTest("localhost")

	// unsecured features
	stm, conn := tUtilStreamInit(r, userRep, blockListRep)
	tUtilStreamOpen(conn)

	elem := conn.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())

	elem = conn.outboundRead()
	require.Equal(t, "stream:features", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("starttls", tlsNamespace))

	require.Equal(t, connected, stm.getState())

	// secured features
	stm2, conn2 := tUtilStreamInit(r, userRep, blockListRep)
	stm2.setSecured(true)

	tUtilStreamOpen(conn2)

	elem = conn2.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())

	elem = conn2.outboundRead()
	require.Equal(t, "stream:features", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("mechanisms", saslNamespace))
}

func TestStream_TLS(t *testing.T) {
	r, userRep, blockListRep := setupTest("localhost")

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r, userRep, blockListRep)
	tUtilStreamOpen(conn)

	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	_, _ = conn.inboundWrite([]byte(`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`))

	elem := conn.outboundRead()

	require.Equal(t, "proceed", elem.Name())
	require.Equal(t, "urn:ietf:params:xml:ns:xmpp-tls", elem.Namespace())

	require.True(t, stm.IsSecured())
}

func TestStream_FailAuthenticate(t *testing.T) {
	r, userRep, blockListRep := setupTest("localhost")

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "user", Password: "pencil"})

	_, conn := tUtilStreamInit(r, userRep, blockListRep)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	// wrong mechanism
	_, _ = conn.inboundWrite([]byte(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="FOO"/>`))

	elem := conn.outboundRead()
	require.Equal(t, "failure", elem.Name())

	_, _ = conn.inboundWrite([]byte(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="PLAIN">AHVzZXIAYQ==</auth>`))

	elem = conn.outboundRead()
	require.Equal(t, "failure", elem.Name())

	// non-SASL
	_, _ = conn.inboundWrite([]byte(`<iq type='set' id='auth2'><query xmlns='jabber:iq:auth'>
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
	r, userRep, blockListRep := setupTest("localhost")

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r, userRep, blockListRep)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	// no method...
	_, _ = conn.inboundWrite([]byte(`<compress xmlns="http://jabber.org/protocol/compress"/>`))
	elem := conn.outboundRead()
	require.Equal(t, "failure", elem.Name())
	require.NotNil(t, elem.Elements().Child("setup-failed"))

	// invalid method...
	_, _ = conn.inboundWrite([]byte(`<compress xmlns="http://jabber.org/protocol/compress">
<method>7z</method>
</compress>`))
	elem = conn.outboundRead()
	require.Equal(t, "failure", elem.Name())
	require.NotNil(t, elem.Elements().Child("unsupported-method"))

	// valid method...
	_, _ = conn.inboundWrite([]byte(`<compress xmlns="http://jabber.org/protocol/compress">
<method>zlib</method>
</compress>`))

	elem = conn.outboundRead()
	require.Equal(t, "compressed", elem.Name())
	require.Equal(t, "http://jabber.org/protocol/compress", elem.Namespace())

	time.Sleep(time.Millisecond * 100) // wait until processed...

	require.True(t, stm.isCompressed())
}

func TestStream_StartSession(t *testing.T) {
	r, userRep, blockListRep := setupTest("localhost")

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r, userRep, blockListRep)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamBind(conn, t)
	tUtilStreamStartSession(conn, t)

	require.Equal(t, bound, stm.getState())
}

func TestStream_SendIQ(t *testing.T) {
	r, userRep, blockListRep := setupTest("localhost")

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r, userRep, blockListRep)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamBind(conn, t)
	tUtilStreamStartSession(conn, t)

	require.Equal(t, bound, stm.getState())

	// request roster...
	iqID := uuid.New().String()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.AppendElement(xmpp.NewElementNamespace("query", "jabber:iq:roster"))

	_, _ = conn.inboundWrite([]byte(iq.String()))

	elem := conn.outboundRead()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
	require.NotNil(t, elem.Elements().ChildNamespace("query", "jabber:iq:roster"))

	requested, _ := stm.Value("roster:requested").(bool)
	require.True(t, requested)
}

func TestStream_SendPresence(t *testing.T) {
	r, userRep, blockListRep := setupTest("localhost")

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r, userRep, blockListRep)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamBind(conn, t)
	tUtilStreamStartSession(conn, t)

	require.Equal(t, bound, stm.getState())

	_, _ = conn.inboundWrite([]byte(`
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
	r, userRep, blockListRep := setupTest("localhost")

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r, userRep, blockListRep)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamBind(conn, t)

	require.Equal(t, bound, stm.getState())

	// define a second stream...
	jFrom, _ := jid.New("user", "localhost", "balcony", true)
	jTo, _ := jid.New("ortuman", "localhost", "garden", true)

	stm2 := stream.NewMockC2S("abcd7890", jTo)
	stm2.SetPresence(xmpp.NewPresence(jTo, jTo, xmpp.AvailableType))

	r.Bind(context.Background(), stm2)

	msgID := uuid.New().String()
	msg := xmpp.NewMessageType(msgID, xmpp.ChatType)
	msg.SetFromJID(jFrom)
	msg.SetToJID(jTo)
	body := xmpp.NewElementName("body")
	body.SetText("Hi buddy!")
	msg.AppendElement(body)

	_, _ = conn.inboundWrite([]byte(msg.String()))

	// to full jid...
	elem := stm2.ReceiveElement()
	require.Equal(t, "message", elem.Name())
	require.Equal(t, msgID, elem.ID())

	// to bare jid...
	msg.SetToJID(jTo.ToBareJID())
	_, _ = conn.inboundWrite([]byte(msg.String()))
	elem = stm2.ReceiveElement()
	require.Equal(t, "message", elem.Name())
	require.Equal(t, msgID, elem.ID())
}

func TestStream_SendToBlockedJID(t *testing.T) {
	r, userRep, blockListRep := setupTest("localhost")

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "user", Password: "pencil"})

	stm, conn := tUtilStreamInit(r, userRep, blockListRep)
	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamAuthenticate(conn, t)

	tUtilStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	tUtilStreamBind(conn, t)
	tUtilStreamStartSession(conn, t)

	require.Equal(t, bound, stm.getState())

	_ = blockListRep.InsertBlockListItem(context.Background(), &model.BlockListItem{
		Username: "user",
		JID:      "hamlet@localhost",
	})

	// send presence to a blocked JID...
	_, _ = conn.inboundWrite([]byte(`<presence to="hamlet@localhost"/>`))

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
	_, _ = conn.inboundWrite([]byte(s))
}

func tUtilStreamAuthenticate(conn *fakeSocketConn, t *testing.T) {
	_, _ = conn.inboundWrite([]byte(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="PLAIN">AHVzZXIAcGVuY2ls</auth>`))

	elem := conn.outboundRead()
	require.Equal(t, "success", elem.Name())
}

func tUtilStreamBind(conn *fakeSocketConn, t *testing.T) {
	// bind a resource
	_, _ = conn.inboundWrite([]byte(`<iq type="set" id="bind_1">
<bind xmlns="urn:ietf:params:xml:ns:xmpp-bind">
<resource>balcony</resource>
</bind>
</iq>`))

	elem := conn.outboundRead()
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().Child("bind"))
}

func tUtilStreamStartSession(conn *fakeSocketConn, t *testing.T) {
	// open session
	_, _ = conn.inboundWrite([]byte(`<iq type="set" id="aab8a">
<session xmlns="urn:ietf:params:xml:ns:xmpp-session"/>
</iq>`))

	elem := conn.outboundRead()
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, xmpp.ResultType, elem.Type())

	time.Sleep(time.Millisecond * 100) // wait until stream internal state changes
}

func tUtilStreamInit(r router.Router, userRep repository.User, blockListRep repository.BlockList) (*inStream, *fakeSocketConn) {
	conn := newFakeSocketConn()
	tr := transport.NewSocketTransport(conn)
	stm := newStream(
		"abcd1234",
		tUtilInStreamDefaultConfig(),
		tr,
		tUtilInitModules(r),
		&component.Components{},
		r,
		userRep,
		blockListRep)
	return stm.(*inStream), conn
}

func tUtilInStreamDefaultConfig() *streamConfig {
	return &streamConfig{
		connectTimeout:   time.Second,
		keepAlive:        time.Second,
		maxStanzaSize:    8192,
		resourceConflict: Reject,
		compression:      CompressConfig{Level: compress.DefaultCompression},
		sasl:             []string{"plain", "digest_md5", "scram_sha_1", "scram_sha_256", "scram_sha_512"},
	}
}

func tUtilInitModules(r router.Router) *module.Modules {
	modules := map[string]struct{}{}
	modules["roster"] = struct{}{}
	modules["blocking_command"] = struct{}{}

	repContainer, _ := storage.New(&storage.Config{Type: storage.Memory})
	return module.New(&module.Config{Enabled: modules}, r, repContainer, "alloc-1234")
}
