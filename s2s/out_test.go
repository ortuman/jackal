/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/module/offline"
	"github.com/ortuman/jackal/module/xep0077"
	"github.com/ortuman/jackal/module/xep0092"
	"github.com/ortuman/jackal/module/xep0199"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xmpp"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestOutStream_Start(t *testing.T) {
	r := setupTest(jackaDomain)

	cfg, _ := tUtilOutStreamDefaultConfig()
	stm := newOutStream(r)
	defer stm.Disconnect(context.Background(), nil)

	// wrong verification name...
	cfg.dbVerify = xmpp.NewElementName("foo")
	err := stm.start(context.Background(), cfg)
	require.NotNil(t, err)

	cfg.dbVerify = nil
	_ = stm.start(context.Background(), cfg)
	err = stm.start(context.Background(), cfg)
	require.NotNil(t, err) // already started
}

func TestOutStream_Disconnect(t *testing.T) {
	r := setupTest(jackaDomain)

	cfg, conn := tUtilOutStreamDefaultConfig()
	stm := newOutStream(r)

	_ = stm.start(context.Background(), cfg)
	stm.Disconnect(context.Background(), nil)
	require.True(t, conn.waitClose())

	require.Equal(t, outDisconnected, stm.getState())
}

func TestOutStream_BadConnect(t *testing.T) {
	r := setupTest(jackaDomain)

	_, conn := tUtilOutStreamInit(t, r)

	// invalid namespace
	_, _ = conn.inboundWriteString(`
<stream:stream xmlns='jabber:client' from='jabber.org' to='jackal.im'>
`)
	require.True(t, conn.waitClose())
}

func TestOutStream_Features(t *testing.T) {
	r := setupTest(jackaDomain)

	_, conn := tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)

	// invalid stanza type...
	_, _ = conn.inboundWriteString(`
<mechanisms/>
`)
	require.True(t, conn.waitClose())

	// invalid namespace...
	_, conn = tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)

	_, _ = conn.inboundWriteString(`
<stream:features/>
`)
	require.True(t, conn.waitClose())

	// invalid version...
	_, conn = tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)

	_, _ = conn.inboundWriteString(`
<stream:features xmlns:stream="http://etherx.jabber.org/streams"/>
`)
	require.True(t, conn.waitClose())

	// starttls not available...
	_, conn = tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	_, _ = conn.inboundWriteString(`
<stream:features xmlns:stream="http://etherx.jabber.org/streams" version="1.0"/>
`)
	require.True(t, conn.waitClose())
}

func TestOutStream_DBVerify(t *testing.T) {
	r := setupTest(jackaDomain)

	cfg, conn := tUtilOutStreamDefaultConfig()
	dbVerify := xmpp.NewElementName("db:verify")
	key := uuid.New()
	dbVerify.SetID("abcde")
	dbVerify.SetFrom("jackal.im")
	dbVerify.SetTo("jabber.org")
	dbVerify.SetText(key)
	cfg.dbVerify = dbVerify

	stm := tUtilOutStreamInitWithConfig(t, r, cfg, conn)
	atomic.StoreUint32(&stm.secured, 1)
	tUtilOutStreamOpen(conn)

	_, _ = conn.inboundWriteString(securedFeatures)
	elem := conn.outboundRead()
	require.Equal(t, "db:verify", elem.Name())
	require.Equal(t, key, elem.Text())

	// unsupported stanza...
	_, _ = conn.inboundWriteString(`
<dbverify/>
`)
	select {
	case sErr := <-stm.done():
		require.Equal(t, "unsupported-stanza-type", sErr.Error())
	case <-time.After(time.Second):
		require.Fail(t, "expecting session error")
	}

	cfg, conn = tUtilOutStreamDefaultConfig()
	cfg.dbVerify = dbVerify
	stm = tUtilOutStreamInitWithConfig(t, r, cfg, conn)
	atomic.StoreUint32(&stm.secured, 1)
	tUtilOutStreamOpen(conn)
	_, _ = conn.inboundWriteString(securedFeatures)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`
<db:verify id='abcde' from='jabber.org' to='jackal.im' type='valid'/>
`)
	select {
	case ok := <-stm.verify():
		require.True(t, ok)
	case <-time.After(time.Second):
		require.Fail(t, "expecting dialback valid verification")
	}
}

func TestOutStream_StartTLS(t *testing.T) {
	r := setupTest(jackaDomain)

	// unsupported stanza...
	_, conn := tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	_, _ = conn.inboundWriteString(unsecuredFeatures)
	elem := conn.outboundRead()
	require.Equal(t, "starttls", elem.Name())
	require.Equal(t, tlsNamespace, elem.Namespace())

	_, _ = conn.inboundWriteString(`<foo/>`)
	require.True(t, conn.waitClose())

	// invalid namespace
	_, conn = tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	_, _ = conn.inboundWriteString(unsecuredFeatures)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`<proceed xmlns="foo"/>`)
	require.True(t, conn.waitClose())

	// valid
	stm, conn := tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	_, _ = conn.inboundWriteString(unsecuredFeatures)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`<proceed xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`)
	_ = conn.outboundRead()

	require.True(t, stm.isSecured())
}

func TestOutStream_Authenticate(t *testing.T) {
	r := setupTest(jackaDomain)

	// unsupported stanza...
	stm, conn := tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	_, _ = conn.inboundWriteString(securedFeaturesWithExternal)

	elem := conn.outboundRead()
	require.Equal(t, "auth", elem.Name())
	require.Equal(t, "urn:ietf:params:xml:ns:xmpp-sasl", elem.Namespace())
	require.Equal(t, "EXTERNAL", elem.Attributes().Get("mechanism"))

	_, _ = conn.inboundWriteString(`
<foo/>
`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	_, _ = conn.inboundWriteString(securedFeaturesWithExternal)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`
<foo xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>
`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	_, _ = conn.inboundWriteString(securedFeaturesWithExternal)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`
<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>
`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	_, _ = conn.inboundWriteString(securedFeaturesWithExternal)
	_ = conn.outboundRead()

	// store pending stanza...
	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.AppendElement(xmpp.NewElementNamespace("query", "jabber:foo"))
	stm.SendElement(context.Background(), iq)

	_, _ = conn.inboundWriteString(`
<success xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>
`)
	elem = conn.outboundRead()
	require.True(t, stm.isAuthenticated())

	tUtilOutStreamOpen(conn)
	_, _ = conn.inboundWriteString(securedFeaturesWithExternal)

	elem = conn.outboundRead() // ...expect receiving pending stanza
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
}

func TestOutStream_Dialback(t *testing.T) {
	r := setupTest(jackaDomain)

	// unsupported stanza...
	stm, conn := tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	_, _ = conn.inboundWriteString(securedFeatures)

	elem := conn.outboundRead()
	require.Equal(t, "db:result", elem.Name())

	// invalid from...
	_, _ = conn.inboundWriteString(`
<db:result from="foo.org"/>
`)
	require.True(t, conn.waitClose())

	// failed
	stm, conn = tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	_, _ = conn.inboundWriteString(securedFeatures)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`
<db:result from="jabber.org" to="jackal.im" type="failed"/>
`)
	require.True(t, conn.waitClose())

	// successful
	stm, conn = tUtilOutStreamInit(t, r)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)

	_, _ = conn.inboundWriteString(securedFeatures)
	_ = conn.outboundRead()

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	stm.SendElement(context.Background(), iq) //...store pending...

	_, _ = conn.inboundWriteString(`
<db:result from="jabber.org" to="jackal.im" type="valid"/>
`)
	elem = conn.outboundRead()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
}

func tUtilOutStreamOpen(conn *fakeSocketConn) {
	// open stream from remote server...
	_, _ = conn.inboundWriteString(`
<?xml version="1.0"?>
<stream:stream xmlns="jabber:server" 
 xmlns:stream="http://etherx.jabber.org/streams" xmlns:db="jabber:server:dialback" 
 from="jabber.org" to="jackal.im" version="1.0">
`)
}

func tUtilOutStreamInitWithConfig(t *testing.T, r *router.Router, cfg *streamConfig, conn *fakeSocketConn) *outStream {
	stm := newOutStream(r)
	_ = stm.start(context.Background(), cfg)

	elem := conn.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())
	require.Equal(t, "jabber:server", elem.Namespace())
	require.Equal(t, "jabber:server:dialback", elem.Attributes().Get("xmlns:db"))
	return stm
}

func tUtilOutStreamInit(t *testing.T, r *router.Router) (*outStream, *fakeSocketConn) {
	cfg, conn := tUtilOutStreamDefaultConfig()
	stm := newOutStream(r)
	_ = stm.start(context.Background(), cfg)

	elem := conn.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())
	require.Equal(t, "jabber:server", elem.Namespace())
	require.Equal(t, "jabber:server:dialback", elem.Attributes().Get("xmlns:db"))
	return stm, conn
}

func tUtilOutStreamDefaultConfig() (*streamConfig, *fakeSocketConn) {
	modules := map[string]struct{}{}
	modules["roster"] = struct{}{}
	modules["last_activity"] = struct{}{}
	modules["private"] = struct{}{}
	modules["vcard"] = struct{}{}
	modules["registration"] = struct{}{}
	modules["version"] = struct{}{}
	modules["ping"] = struct{}{}
	modules["blocking_command"] = struct{}{}
	modules["offline"] = struct{}{}

	conn := newFakeSocketConn()
	tr := transport.NewSocketTransport(conn, 4096)
	return &streamConfig{
		remoteDomain: "jabber.org",
		modConfig: &module.Config{
			Enabled:      modules,
			Offline:      offline.Config{QueueSize: 10},
			Registration: xep0077.Config{AllowRegistration: true, AllowChange: true},
			Version:      xep0092.Config{ShowOS: true},
			Ping:         xep0199.Config{SendInterval: 5, Send: true},
		},
		connectTimeout: time.Second,
		transport:      tr,
		maxStanzaSize:  8192,
		keyGen:         &keyGen{secret: "s3cr3t"},
	}, conn
}
