/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"github.com/ortuman/jackal/router/host"
	"github.com/ortuman/jackal/xmpp"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestOutStream_Disconnect(t *testing.T) {
	h := setupTestHosts(jackaDomain)

	cfg, dialer, conn := tUtilOutStreamDefaultConfig()
	stm := newOutStream(cfg, h, dialer)
	_ = stm.reconnect(context.Background())

	stm.Disconnect(context.Background(), nil)
	require.True(t, conn.waitClose())

	require.Equal(t, outDisconnected, stm.getState())
}

func TestOutStream_BadConnect(t *testing.T) {
	h := setupTestHosts(jackaDomain)

	_, conn := tUtilOutStreamInit(t, h)

	// invalid namespace
	_, _ = conn.inboundWriteString(`
<stream:stream xmlns='jabber:client' from='jabber.org' to='jackal.im'>
`)
	require.True(t, conn.waitClose())
}

func TestOutStream_Features(t *testing.T) {
	h := setupTestHosts(jackaDomain)

	_, conn := tUtilOutStreamInit(t, h)
	tUtilOutStreamOpen(conn)

	// invalid stanza type...
	_, _ = conn.inboundWriteString(`
<mechanisms/>
`)
	require.True(t, conn.waitClose())

	// invalid namespace...
	_, conn = tUtilOutStreamInit(t, h)
	tUtilOutStreamOpen(conn)

	_, _ = conn.inboundWriteString(`
<stream:features/>
`)
	require.True(t, conn.waitClose())

	// invalid version...
	_, conn = tUtilOutStreamInit(t, h)
	tUtilOutStreamOpen(conn)

	_, _ = conn.inboundWriteString(`
<stream:features xmlns:stream="http://etherx.jabber.org/streams"/>
`)
	require.True(t, conn.waitClose())

	// starttls not available...
	_, conn = tUtilOutStreamInit(t, h)
	tUtilOutStreamOpen(conn)
	_, _ = conn.inboundWriteString(`
<stream:features xmlns:stream="http://etherx.jabber.org/streams" version="1.0"/>
`)
	require.True(t, conn.waitClose())
}

func TestOutStream_StartTLS(t *testing.T) {
	h := setupTestHosts(jackaDomain)

	// unsupported stanza...
	_, conn := tUtilOutStreamInit(t, h)
	tUtilOutStreamOpen(conn)
	_, _ = conn.inboundWriteString(unsecuredFeatures)
	elem := conn.outboundRead()
	require.Equal(t, "starttls", elem.Name())
	require.Equal(t, tlsNamespace, elem.Namespace())

	_, _ = conn.inboundWriteString(`<foo/>`)
	require.True(t, conn.waitClose())

	// invalid namespace
	_, conn = tUtilOutStreamInit(t, h)
	tUtilOutStreamOpen(conn)
	_, _ = conn.inboundWriteString(unsecuredFeatures)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`<proceed xmlns="foo"/>`)
	require.True(t, conn.waitClose())

	// valid
	stm, conn := tUtilOutStreamInit(t, h)
	tUtilOutStreamOpen(conn)
	_, _ = conn.inboundWriteString(unsecuredFeatures)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`<proceed xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`)
	_ = conn.outboundRead()

	require.True(t, stm.isSecured())
}

func TestOutStream_Authenticate(t *testing.T) {
	h := setupTestHosts(jackaDomain)

	// unsupported stanza...
	stm, conn := tUtilOutStreamInit(t, h)
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

	stm, conn = tUtilOutStreamInit(t, h)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	_, _ = conn.inboundWriteString(securedFeaturesWithExternal)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`
<foo xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>
`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilOutStreamInit(t, h)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	_, _ = conn.inboundWriteString(securedFeaturesWithExternal)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`
<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>
`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilOutStreamInit(t, h)
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
	h := setupTestHosts(jackaDomain)

	// unsupported stanza...
	stm, conn := tUtilOutStreamInit(t, h)
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
	stm, conn = tUtilOutStreamInit(t, h)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	_, _ = conn.inboundWriteString(securedFeatures)
	_ = conn.outboundRead()

	_, _ = conn.inboundWriteString(`
<db:result from="jabber.org" to="jackal.im" type="failed"/>
`)
	require.True(t, conn.waitClose())

	// successful
	stm, conn = tUtilOutStreamInit(t, h)
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

func tUtilOutStreamInitWithConfig(t *testing.T, hosts *host.Hosts, cfg *outConfig, conn *fakeSocketConn) *outStream {
	d := newDialer()
	d.dialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
		return conn, nil
	}
	stm := newOutStream(cfg, hosts, d)
	_ = stm.reconnect(context.Background()) // start stream

	elem := conn.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())
	require.Equal(t, "jabber:server", elem.Namespace())
	require.Equal(t, "jabber:server:dialback", elem.Attributes().Get("xmlns:db"))
	return stm
}

func tUtilOutStreamInit(t *testing.T, hosts *host.Hosts) (*outStream, *fakeSocketConn) {
	cfg, dialer, conn := tUtilOutStreamDefaultConfig()
	stm := newOutStream(cfg, hosts, dialer)
	_ = stm.reconnect(context.Background()) // start stream

	elem := conn.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())
	require.Equal(t, "jabber:server", elem.Namespace())
	require.Equal(t, "jabber:server:dialback", elem.Attributes().Get("xmlns:db"))
	return stm, conn
}

func tUtilOutStreamDefaultConfig() (*outConfig, Dialer, *fakeSocketConn) {
	conn := newFakeSocketConn()
	d := newDialer()
	d.dialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
		return conn, nil
	}
	return &outConfig{
		remoteDomain:  "jabber.org",
		maxStanzaSize: 8192,
		keepAlive:     4096,
		keyGen:        &keyGen{secret: "s3cr3t"},
	}, d, conn
}
