/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/xml"
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

func (c *fakeSocketConn) outboundRead() xml.XElement {
	var elem xml.XElement
	var err error
	p := xml.NewParser(c.wr, xml.SocketStream, 0)
	for err == nil {
		elem, err = p.ParseElement()
		if elem != nil {
			return elem
		}
	}
	return &xml.Element{}
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
