/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	utiltls "github.com/ortuman/jackal/util/tls"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestStream_ConnectTimeout(t *testing.T) {
	r, h := setupTestRouter(jackaDomain)

	op := NewOutProvider(&Config{KeepAlive: time.Second}, h)

	stm, _ := tUtilInStreamInit(t, r, op, false)
	time.Sleep(time.Millisecond * 1500)
	require.Equal(t, inDisconnected, stm.getState())
}

func TestStream_Disconnect(t *testing.T) {
	r, h := setupTestRouter(jackaDomain)

	op := NewOutProvider(&Config{}, h)

	stm, conn := tUtilInStreamInit(t, r, op, false)
	stm.Disconnect(context.Background(), nil)
	require.True(t, conn.waitClose())

	require.Equal(t, inDisconnected, stm.getState())
}

func TestStream_Features(t *testing.T) {
	r, h := setupTestRouter(jackaDomain)

	op := NewOutProvider(&Config{KeepAlive: time.Second}, h)

	// unsecured features
	stm, conn := tUtilInStreamInit(t, r, op, false)
	tUtilInStreamOpen(conn)

	elem := conn.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())

	elem = conn.outboundRead()
	require.Equal(t, "stream:features", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("starttls", tlsNamespace))
	require.Equal(t, inConnected, stm.getState())

	// secured features
	stm, conn = tUtilInStreamInit(t, r, op, false)
	atomic.StoreUint32(&stm.secured, 1)
	tUtilInStreamOpen(conn)

	elem = conn.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())

	elem = conn.outboundRead()
	require.NotNil(t, elem.Elements().ChildNamespace("mechanisms", saslNamespace))
	require.NotNil(t, elem.Elements().ChildNamespace("dialback", dialbackNamespace))
	require.Equal(t, inConnected, stm.getState())

	// secured features (authenticated)
	stm, conn = tUtilInStreamInit(t, r, op, false)
	atomic.StoreUint32(&stm.secured, 1)
	atomic.StoreUint32(&stm.authenticated, 1)
	tUtilInStreamOpen(conn)

	elem = conn.outboundRead()
	require.Equal(t, "stream:stream", elem.Name())

	elem = conn.outboundRead()
	require.Nil(t, elem.Elements().ChildNamespace("mechanisms", saslNamespace))
	require.NotNil(t, elem.Elements().ChildNamespace("dialback", dialbackNamespace))
	require.Equal(t, inConnected, stm.getState())
}

func TestStream_TLS(t *testing.T) {
	r, h := setupTestRouter(jackaDomain)

	op := NewOutProvider(&Config{KeepAlive: time.Second}, h)

	stm, conn := tUtilInStreamInit(t, r, op, false)
	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	// wrong namespace...
	_, _ = conn.inboundWriteString(`<starttls xmlns="foo:ns"/>`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilInStreamInit(t, r, op, false)
	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	// wrong name...
	_, _ = conn.inboundWriteString(`<foo xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilInStreamInit(t, r, op, false)
	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...

	_, _ = conn.inboundWriteString(`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>`)

	elem := conn.outboundRead()

	require.Equal(t, "proceed", elem.Name())
	require.Equal(t, "urn:ietf:params:xml:ns:xmpp-tls", elem.Namespace())

	require.True(t, stm.isSecured())
}

func TestStream_Authenticate(t *testing.T) {
	r, h := setupTestRouter(jackaDomain)

	op := NewOutProvider(&Config{KeepAlive: time.Second}, h)

	stm, conn := tUtilInStreamInit(t, r, op, false)
	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...
	atomic.StoreUint32(&stm.secured, 1)

	// invalid namespace...
	_, _ = conn.inboundWriteString(`<auth xmlns="foo:ns" mechanism="EXTERNAL">=</auth>`)
	require.True(t, conn.waitClose())

	stm, conn = tUtilInStreamInit(t, r, op, true)
	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...
	atomic.StoreUint32(&stm.secured, 1)

	// failed peer certificate...
	stm, conn = tUtilInStreamInit(t, r, op, false)
	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...
	atomic.StoreUint32(&stm.secured, 1)

	_, _ = conn.inboundWriteString(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="EXTERNAL">=</auth>`)
	elem := conn.outboundRead()
	require.Equal(t, "failure", elem.Name())
	require.Equal(t, saslNamespace, elem.Namespace())

	// invalid mechanism...
	stm, conn = tUtilInStreamInit(t, r, op, true)
	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...
	atomic.StoreUint32(&stm.secured, 1)

	_, _ = conn.inboundWriteString(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="FOO">=</auth>`)
	elem = conn.outboundRead()
	require.Equal(t, "failure", elem.Name())
	require.Equal(t, saslNamespace, elem.Namespace())

	// valid auth...
	_, _ = conn.inboundWriteString(`<auth xmlns="urn:ietf:params:xml:ns:xmpp-sasl" mechanism="EXTERNAL">=</auth>`)
	elem = conn.outboundRead()
	require.Equal(t, "success", elem.Name())
	require.Equal(t, saslNamespace, elem.Namespace())
}

func TestStream_DialbackVerify(t *testing.T) {
	r, h := setupTestRouter(jackaDomain)

	op := NewOutProvider(&Config{KeepAlive: time.Second}, h)

	stm, conn := tUtilInStreamInit(t, r, op, false)
	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...
	atomic.StoreUint32(&stm.secured, 1)
	atomic.StoreUint32(&stm.authenticated, 1)

	// invalid host
	_, _ = conn.inboundWriteString(`<db:verify id="abcde" from="localhost" to="foo.org">abcd</db:verify>`)
	elem := conn.outboundRead()
	require.Equal(t, "db:verify", elem.Name())
	require.NotNil(t, elem.Elements().Child("error"))
	require.NotNil(t, elem.Elements().Child("error").Elements().Child("item-not-found"))

	// invalid key
	_, _ = conn.inboundWriteString(`<db:verify id="abcde" from="localhost" to="jackal.im">abcd</db:verify>`)
	elem = conn.outboundRead()
	require.Equal(t, "db:verify", elem.Name())
	require.Equal(t, "invalid", elem.Type())

	// valid key
	kg := &keyGen{secret: "s3cr3t"}
	key := kg.generate("localhost", "jackal.im", "abcde")

	_, _ = conn.inboundWriteString(fmt.Sprintf(`<db:verify id="abcde" from="localhost" to="jackal.im">%s</db:verify>`, key))
	elem = conn.outboundRead()
	require.Equal(t, "db:verify", elem.Name())
	require.Equal(t, "valid", elem.Type())
}

func TestStream_DialbackAuthorize(t *testing.T) {
	r, h := setupTestRouter(jackaDomain)

	op := NewOutProvider(&Config{KeepAlive: time.Second}, h)
	op.dialer.(*dialer).srvResolve = func(_, _, _ string) (cname string, addrs []*net.SRV, err error) {
		return "", []*net.SRV{{Target: "jackal.im", Port: 5269}}, nil
	}

	stm, conn := tUtilInStreamInit(t, r, op, false)
	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...
	atomic.StoreUint32(&stm.secured, 1)
	atomic.StoreUint32(&stm.authenticated, 1)

	_, _ = conn.inboundWriteString(`<db:result to="foo.org">abcd</db:result>`)
	elem := conn.outboundRead()
	require.Equal(t, "db:result", elem.Name())
	require.Equal(t, xmpp.ErrorType, elem.Type())
	require.NotNil(t, elem.Elements().Child("error"))
	require.NotNil(t, elem.Elements().Child("error").Elements().Child("item-not-found"))

	cfg, tr, conn := tUtilInStreamDefaultConfig(t, false)
	outConn := newFakeSocketConn()
	op.dialer.(*dialer).dialContext = func(ctx context.Context, network, address string) (conn net.Conn, err error) {
		return outConn, nil
	}
	stm = newInStream(cfg, tr, &module.Modules{}, op.newOut, r)

	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...
	atomic.StoreUint32(&stm.secured, 1)
	atomic.StoreUint32(&stm.authenticated, 1)

	_, _ = conn.inboundWriteString(`<db:result to="jackal.im">abcd</db:result>`)
	_ = outConn.Close()
	elem = conn.outboundRead()
	require.Equal(t, "db:result", elem.Name())
	require.Equal(t, xmpp.ErrorType, elem.Type())
	require.NotNil(t, elem.Elements().Child("error"))
	require.NotNil(t, elem.Elements().Child("error").Elements().Child("remote-server-timeout"))

	// authorize dialback key
	cfg, tr, conn = tUtilInStreamDefaultConfig(t, false)
	outConn = newFakeSocketConn()
	op.dialer.(*dialer).dialContext = func(ctx context.Context, network, address string) (conn net.Conn, err error) {
		return outConn, nil
	}
	stm = newInStream(cfg, tr, &module.Modules{}, op.newOut, r)

	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...
	atomic.StoreUint32(&stm.secured, 1)
	atomic.StoreUint32(&stm.authenticated, 1)

	_, _ = conn.inboundWriteString(`<db:result to="jackal.im">abcd</db:result>`)

	_, _ = outConn.inboundWriteString(`
<?xml version="1.0"?>
<stream:stream xmlns="jabber:server"
 xmlns:stream="http://etherx.jabber.org/streams" xmlns:db="jabber:server:dialback"
 from="jackal.im" version="1.0">
<stream:features>
 <starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls">
   <required/>
 </starttls>
</stream:features>
`)
	_ = outConn.outboundRead() // stream:stream
	_ = outConn.outboundRead() // starttls

	_, _ = outConn.inboundWriteString(`
<proceed xmlns="urn:ietf:params:xml:ns:xmpp-tls"/>
`)
	_ = outConn.outboundRead() // stream:stream

	_, _ = outConn.inboundWriteString(`
<?xml version="1.0"?>
<stream:stream xmlns="jabber:server"
 xmlns:stream="http://etherx.jabber.org/streams" xmlns:db="jabber:server:dialback"
 from="jackal.im" version="1.0">
<stream:features>
  <dialback xmlns="urn:xmpp:features:dialback">
   <errors/>
  </dialback>
</stream:features>
`)
	_ = outConn.outboundRead() // db:verify

	_, _ = outConn.inboundWriteString(`
<db:verify from="jackal.im" type="valid"/>
`)
	elem = conn.outboundRead()
	require.Equal(t, "db:result", elem.Name())
	require.Equal(t, "valid", elem.Type())
}

func TestStream_SendElement(t *testing.T) {
	r, h := setupTestRouter(jackaDomain)

	op := NewOutProvider(&Config{KeepAlive: time.Second}, h)

	fromJID, _ := jid.New("ortuman", "localhost", "garden", true)
	toJID, _ := jid.New("ortuman", "jackal.im", "garden", true)

	stm2 := stream.NewMockC2S("abcd7890", toJID)
	stm2.SetPresence(xmpp.NewPresence(toJID, toJID, xmpp.AvailableType))

	r.Bind(context.Background(), stm2)

	stm, conn := tUtilInStreamInit(t, r, op, false)
	tUtilInStreamOpen(conn)
	_ = conn.outboundRead() // read stream opening...
	_ = conn.outboundRead() // read stream features...
	atomic.StoreUint32(&stm.secured, 1)
	atomic.StoreUint32(&stm.authenticated, 1)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.ResultType)
	iq.SetFromJID(fromJID)
	iq.SetToJID(toJID)
	_, _ = conn.inboundWriteString(iq.String())

	elem := stm2.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	// invalid from...
	iq.SetFrom("foo.org")
	_, _ = conn.inboundWriteString(iq.String())
	require.True(t, conn.waitClose())
}

func tUtilInStreamInit(t *testing.T, router router.Router, outProvider *OutProvider, loadPeerCertificate bool) (*inStream, *fakeSocketConn) {
	cfg, tr, conn := tUtilInStreamDefaultConfig(t, loadPeerCertificate)
	stm := newInStream(cfg, tr, &module.Modules{}, outProvider.newOut, router)
	return stm, conn
}

func tUtilInStreamOpen(conn *fakeSocketConn) {
	s := `<?xml version="1.0"?>
	<stream:stream xmlns:stream="http://etherx.jabber.org/streams"
	version="1.0" xmlns="jabber:server" to="jackal.im" from="localhost" xmlns:xml="http://www.w3.org/XML/1998/namespace">
`
	_, _ = conn.inboundWriteString(s)
}

func tUtilInStreamDefaultConfig(t *testing.T, loadPeerCertificate bool) (*inConfig, transport.Transport, *fakeSocketConn) {
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

	certFile := "../testdata/cert/test.server.crt"
	certKey := "../testdata/cert/test.server.key"
	cer, err := utiltls.LoadCertificate(certKey, certFile, "localhost")
	require.Nil(t, err)

	var peerCerts []*x509.Certificate
	if loadPeerCertificate {
		for _, asn1Data := range cer.Certificate {
			cr, err := x509.ParseCertificate(asn1Data)
			require.Nil(t, err)
			cr.DNSNames = []string{"localhost"}
			peerCerts = append(peerCerts, cr)
		}
	}

	conn := newFakeSocketConnWithPeerCerts(peerCerts)
	tr := transport.NewSocketTransport(conn)
	return &inConfig{
		connectTimeout: time.Second,
		keepAlive:      time.Second,
		maxStanzaSize:  8192,
		keyGen:         &keyGen{secret: "s3cr3t"},
	}, tr, conn
}
