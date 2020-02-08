/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/router/host"

	"github.com/ortuman/jackal/xmpp"
)

const jackaDomain = "jackal.im"

const unsecuredFeatures = `
<stream:features xmlns:stream="http://etherx.jabber.org/streams" version="1.0">
  <starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls">
    <required/>
  </starttls>
</stream:features>
`

const securedFeaturesWithExternal = `
<stream:features xmlns:stream="http://etherx.jabber.org/streams" version="1.0">
  <mechanisms xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><mechanism>EXTERNAL</mechanism></mechanisms>
  <dialback xmlns="urn:xmpp:features:dialback"><errors/></dialback>
</stream:features>
`

const securedFeatures = `
<stream:features xmlns:stream="http://etherx.jabber.org/streams" version="1.0">
  <dialback xmlns="urn:xmpp:features:dialback"><errors/></dialback>
</stream:features>
`

var errFakeSockAlreadyClosed = errors.New("fakeSockReaderWriter: already closed")

type fakeSockReaderWriter struct {
	r      *io.PipeReader
	w      *io.PipeWriter
	closed uint32
}

func newFakeSockReaderWriter() *fakeSockReaderWriter {
	pr, pw := io.Pipe()
	frw := &fakeSockReaderWriter{r: pr, w: pw}
	return frw
}

func (frw *fakeSockReaderWriter) Write(b []byte) (n int, err error) {
	return frw.w.Write(b)
}

func (frw *fakeSockReaderWriter) Read(b []byte) (n int, err error) {
	return frw.r.Read(b)
}

func (frw *fakeSockReaderWriter) Close() error {
	_ = frw.w.Close()
	_ = frw.r.Close()
	return nil
}

type fakeSocketConn struct {
	rd        *fakeSockReaderWriter
	wr        *fakeSockReaderWriter
	wrCh      chan []byte
	p         *xmpp.Parser
	closeCh   chan struct{}
	closed    uint32
	peerCerts []*x509.Certificate
}

func newFakeSocketConn() *fakeSocketConn {
	return newFakeSocketConnWithPeerCerts(nil)
}

func newFakeSocketConnWithPeerCerts(peerCerts []*x509.Certificate) *fakeSocketConn {
	fc := &fakeSocketConn{
		rd:        newFakeSockReaderWriter(),
		wr:        newFakeSockReaderWriter(),
		wrCh:      make(chan []byte, 256),
		closeCh:   make(chan struct{}, 1),
		peerCerts: peerCerts,
	}
	fc.p = xmpp.NewParser(fc.wr, xmpp.SocketStream, 0)
	go fc.loop()
	return fc
}

func (c *fakeSocketConn) Read(b []byte) (n int, err error) {
	if atomic.LoadUint32(&c.closed) == 1 {
		return 0, errFakeSockAlreadyClosed
	}
	return c.rd.Read(b)
}

func (c *fakeSocketConn) Write(b []byte) (n int, err error) {
	if atomic.LoadUint32(&c.closed) == 1 {
		return 0, errFakeSockAlreadyClosed
	}
	wb := make([]byte, len(b))
	copy(wb, b)
	c.wrCh <- wb
	return len(wb), nil
}

func (c *fakeSocketConn) Close() error {
	if atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		_ = c.wr.Close()
		_ = c.rd.Close()
		close(c.closeCh)
		return nil
	}
	return errFakeSockAlreadyClosed
}

func (c *fakeSocketConn) LocalAddr() net.Addr                { return localAddr }
func (c *fakeSocketConn) RemoteAddr() net.Addr               { return remoteAddr }
func (c *fakeSocketConn) SetDeadline(_ time.Time) error      { return nil }
func (c *fakeSocketConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *fakeSocketConn) SetWriteDeadline(_ time.Time) error { return nil }

func (c *fakeSocketConn) ConnectionState() tls.ConnectionState {
	st := tls.ConnectionState{}
	if len(c.peerCerts) > 0 {
		st.PeerCertificates = c.peerCerts
	}
	return st
}

func (c *fakeSocketConn) inboundWriteString(s string) (n int, err error) { return c.rd.Write([]byte(s)) }
func (c *fakeSocketConn) inboundWrite(b []byte) (n int, err error)       { return c.rd.Write(b) }

func (c *fakeSocketConn) outboundRead() xmpp.XElement {
	var elem xmpp.XElement
	var err error
	for err == nil {
		elem, err = c.p.ParseElement()
		if elem != nil {
			return elem
		}
	}
	return &xmpp.Element{}
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
			_, _ = c.wr.Write(b)
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

func setupTest(domain string) *host.Hosts {
	hosts, _ := host.New([]host.Config{{Name: domain, Certificate: tls.Certificate{}}})
	return hosts
}

type fakeS2SServer struct {
	startCh    chan struct{}
	shutdownCh chan struct{}
}

func newFakeS2SServer() *fakeS2SServer {
	return &fakeS2SServer{
		startCh:    make(chan struct{}, 1),
		shutdownCh: make(chan struct{}, 1),
	}
}

func (s *fakeS2SServer) start() {
	s.startCh <- struct{}{}
}

func (s *fakeS2SServer) shutdown(_ context.Context) error {
	s.shutdownCh <- struct{}{}
	return nil
}
