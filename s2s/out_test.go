/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/module/offline"
	"github.com/ortuman/jackal/module/xep0077"
	"github.com/ortuman/jackal/module/xep0092"
	"github.com/ortuman/jackal/module/xep0199"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestOutStream_Start(t *testing.T) {
	cfg, _ := tUtilOutStreamDefaultConfig()
	stm := newOutStream()
	defer stm.Disconnect(nil)

	// wrong verification name...
	cfg.dbVerify = xml.NewElementName("foo")
	err := stm.start(cfg)
	require.NotNil(t, err)

	cfg.dbVerify = nil
	stm.start(cfg)
	err = stm.start(cfg)
	require.NotNil(t, err) // already started
}

func TestOutStream_Disconnect(t *testing.T) {
	cfg, conn := tUtilOutStreamDefaultConfig()
	stm := newOutStream()
	stm.start(cfg)
	stm.Disconnect(nil)
	require.True(t, conn.waitClose())

	require.Equal(t, outDisconnected, stm.getState())
}

func TestOutStream_BadConnect(t *testing.T) {
	_, conn := tUtilOutStreamInit(t)

	// invalid namespace
	conn.inboundWriteString(`
<stream:stream xmlns='jabber:client' from='jabber.org' to='jackal.im'>
`)
	require.True(t, conn.waitClose())
}

func TestOutStream_Features(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	defer host.Shutdown()

	_, conn := tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)

	// invalid stanza type...
	conn.inboundWriteString(`
<mechanisms/>
`)
	require.True(t, conn.waitClose())

	// invalid namespace...
	_, conn = tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	conn.inboundWriteString(`
<stream:features/>
`)
	require.True(t, conn.waitClose())

	// invalid version...
	_, conn = tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	conn.inboundWriteString(`
<stream:features xmlns:stream="http://etherx.jabber.org/streams"/>
`)
	require.True(t, conn.waitClose())

	// starttls not available...
	_, conn = tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	conn.inboundWriteString(`
<stream:features xmlns:stream="http://etherx.jabber.org/streams" version="1.0"/>
`)
	require.True(t, conn.waitClose())
}

func TestOutStream_DBVerify(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	defer host.Shutdown()

	cfg, conn := tUtilOutStreamDefaultConfig()
	dbVerify := xml.NewElementName("db:verify")
	key := uuid.New()
	dbVerify.SetID("abcde")
	dbVerify.SetFrom("jackal.im")
	dbVerify.SetTo("jabber.org")
	dbVerify.SetText(key)
	cfg.dbVerify = dbVerify

	stm := tUtilOutStreamInitWithConfig(t, cfg, conn)
	atomic.StoreUint32(&stm.secured, 1)
	tUtilOutStreamOpen(conn)

	conn.inboundWriteString(securedFeatures)
	elem := conn.outboundRead()
	require.Equal(t, "db:verify", elem.Name())
	require.Equal(t, key, elem.Text())

	// unsupported stanza...
	conn.inboundWriteString(`
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
	stm = tUtilOutStreamInitWithConfig(t, cfg, conn)
	atomic.StoreUint32(&stm.secured, 1)
	tUtilOutStreamOpen(conn)
	conn.inboundWriteString(securedFeatures)
	_ = conn.outboundRead()

	conn.inboundWriteString(`
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
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	defer host.Shutdown()

	// unsupported stanza...
	_, conn := tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	conn.inboundWriteString(unsecuredFeatures)
	elem := conn.outboundRead()
	require.Equal(t, "starttls", elem.Name())
	require.Equal(t, tlsNamespace, elem.Namespace())

	conn.inboundWriteString(`<foo/>`)
	require.True(t, conn.waitClose())

	// invalid namespace
	_, conn = tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	conn.inboundWriteString(unsecuredFeatures)
	_ = conn.outboundRead()

	conn.inboundWriteString(`<proceed xmlns="foo"/>`)
	require.True(t, conn.waitClose())

	// valid
	stm, conn := tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	conn.inboundWriteString(unsecuredFeatures)
	_ = conn.outboundRead()

	conn.inboundWriteString(`<proceed xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`)
	_ = conn.outboundRead()

	require.True(t, stm.isSecured())
}

func TestOutStream_Authenticate(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	defer host.Shutdown()

	// unsupported stanza...
	stm, conn := tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	conn.inboundWriteString(securedFeaturesWithExternal)

	elem := conn.outboundRead()
	require.Equal(t, "auth", elem.Name())
	require.Equal(t, "urn:ietf:params:xml:ns:xmpp-sasl", elem.Namespace())
	require.Equal(t, "EXTERNAL", elem.Attributes().Get("mechanism"))

	conn.inboundWriteString(`
<foo/>
`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	conn.inboundWriteString(securedFeaturesWithExternal)
	_ = conn.outboundRead()

	conn.inboundWriteString(`
<foo xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>
`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	conn.inboundWriteString(securedFeaturesWithExternal)
	_ = conn.outboundRead()

	conn.inboundWriteString(`
<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>
`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	conn.inboundWriteString(securedFeaturesWithExternal)
	_ = conn.outboundRead()

	// store pending stanza...
	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.GetType)
	iq.AppendElement(xml.NewElementNamespace("query", "jabber:foo"))
	stm.SendElement(iq)

	conn.inboundWriteString(`
<success xmlns="urn:ietf:params:xml:ns:xmpp-sasl"/>
`)
	elem = conn.outboundRead()
	require.True(t, stm.isAuthenticated())

	tUtilOutStreamOpen(conn)
	conn.inboundWriteString(securedFeaturesWithExternal)

	elem = conn.outboundRead() // ...expect receiving pending stanza
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
}

func TestOutStream_Dialback(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	defer host.Shutdown()

	// unsupported stanza...
	stm, conn := tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	conn.inboundWriteString(securedFeatures)

	elem := conn.outboundRead()
	require.Equal(t, "db:result", elem.Name())

	// invalid from...
	conn.inboundWriteString(`
<db:result from="foo.org"/>
`)
	require.True(t, conn.waitClose())

	// failed
	stm, conn = tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)
	conn.inboundWriteString(securedFeatures)
	_ = conn.outboundRead()

	conn.inboundWriteString(`
<db:result from="jabber.org" to="jackal.im" type="failed"/>
`)
	require.True(t, conn.waitClose())

	// successful
	stm, conn = tUtilOutStreamInit(t)
	tUtilOutStreamOpen(conn)
	atomic.StoreUint32(&stm.secured, 1)

	conn.inboundWriteString(securedFeatures)
	_ = conn.outboundRead()

	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.GetType)
	stm.SendElement(iq) //...store pending...

	conn.inboundWriteString(`
<db:result from="jabber.org" to="jackal.im" type="valid"/>
`)
	elem = conn.outboundRead()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
}

func tUtilOutStreamOpen(conn *fakeSocketConn) {
	// open stream from remote server...
	conn.inboundWriteString(`
<?xml version="1.0"?>
<stream:stream xmlns="jabber:server" 
 xmlns:stream="http://etherx.jabber.org/streams" xmlns:db="jabber:server:dialback" 
 from="jabber.org" to="jackal.im" version="1.0">
`)
}

func tUtilOutStreamInitWithConfig(t *testing.T, cfg *streamConfig, conn *fakeSocketConn) *outStream {
	stm := newOutStream()
	stm.start(cfg)

	elem := conn.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())
	require.Equal(t, "jabber:server", elem.Namespace())
	require.Equal(t, "jabber:server:dialback", elem.Attributes().Get("xmlns:db"))
	return stm
}

func tUtilOutStreamInit(t *testing.T) (*outStream, *fakeSocketConn) {
	cfg, conn := tUtilOutStreamDefaultConfig()
	stm := newOutStream()
	stm.start(cfg)

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
