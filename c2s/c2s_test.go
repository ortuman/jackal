/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

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
	frw.w.Close()
	frw.r.Close()
	return nil
}

type fakeSocketConn struct {
	rd      *fakeSockReaderWriter
	wr      *fakeSockReaderWriter
	p       *xmpp.Parser
	wrCh    chan []byte
	closeCh chan struct{}
	closed  uint32
}

func newFakeSocketConn() *fakeSocketConn {
	fc := &fakeSocketConn{
		rd:      newFakeSockReaderWriter(),
		wr:      newFakeSockReaderWriter(),
		wrCh:    make(chan []byte, 256),
		closeCh: make(chan struct{}, 1),
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
		c.wr.Close()
		c.rd.Close()
		close(c.closeCh)
		return nil
	}
	return errFakeSockAlreadyClosed
}

func (c *fakeSocketConn) LocalAddr() net.Addr                { return localAddr }
func (c *fakeSocketConn) RemoteAddr() net.Addr               { return remoteAddr }
func (c *fakeSocketConn) SetDeadline(t time.Time) error      { return nil }
func (c *fakeSocketConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *fakeSocketConn) SetWriteDeadline(t time.Time) error { return nil }

func (c *fakeSocketConn) inboundWrite(b []byte) (n int, err error) {
	return c.rd.Write(b)
}

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

func setupTest(domain string) (*router.Router, *memstorage.Storage, func()) {
	r, _ := router.New(&router.Config{
		Hosts: []router.HostConfig{{Name: domain, Certificate: tls.Certificate{}}},
	})
	s := memstorage.New()
	storage.Set(s)
	return r, s, func() {
		storage.Unset()
	}
}

type fakeC2SServer struct {
	startCh    chan struct{}
	shutdownCh chan struct{}
}

func newFakeC2SServer() *fakeC2SServer {
	return &fakeC2SServer{
		startCh:    make(chan struct{}, 1),
		shutdownCh: make(chan struct{}, 1),
	}
}

func (s *fakeC2SServer) start() {
	s.startCh <- struct{}{}
}

func (s *fakeC2SServer) shutdown(ctx context.Context) error {
	s.shutdownCh <- struct{}{}
	return nil
}

func TestC2S_StartAndShutdown(t *testing.T) {
	c2s, fakeSrv := setupTestC2S()

	c2s.Start()
	select {
	case <-fakeSrv.startCh:
		break
	case <-time.After(time.Millisecond * 250):
		require.Fail(t, "c2s start timeout")
	}

	c2s.Shutdown(context.Background())
	select {
	case <-fakeSrv.shutdownCh:
		break
	case <-time.After(time.Millisecond * 250):
		require.Fail(t, "c2s shutdown timeout")
	}
}

func setupTestC2S() (*C2S, *fakeC2SServer) {
	srv := newFakeC2SServer()
	createC2SServer = func(_ *Config, _ *module.Modules, _ *component.Components, _ *router.Router) c2sServer {
		return srv
	}
	c2s, _ := New([]Config{{}}, &module.Modules{}, &component.Components{}, &router.Router{})
	return c2s, srv
}
