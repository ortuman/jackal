/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bytes"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/xml"
)

const mockConnNetwork = "tcp"

const (
	mockConnLocalAddr  = "10.188.17.228"
	mockConnRemoteAddr = "77.230.105.223"
)

type mockConnPipe struct {
	w *io.PipeWriter
	r *io.PipeReader
}

func newMockConnPipe() *mockConnPipe {
	r, w := io.Pipe()
	return &mockConnPipe{w: w, r: r}
}

type mockConnAddress struct {
	network string
	str     string
}

func (ma *mockConnAddress) Network() string {
	return ma.network
}

func (ma *mockConnAddress) String() string {
	return ma.str
}

// MockConn represents a net.Conn mocked implementation.
type MockConn struct {
	r       *bytes.Reader
	p       *xml.Parser
	clPipe  *mockConnPipe
	srvPipe *mockConnPipe
	closed  uint32
	discCh  chan bool
	writeCh chan []byte
	readCh  chan []byte
}

// NewMockConn returns a new initialized MockConn instance.
func NewMockConn() *MockConn {
	mc := &MockConn{
		clPipe:  newMockConnPipe(),
		srvPipe: newMockConnPipe(),
		discCh:  make(chan bool, 1),
		writeCh: make(chan []byte, 1),
		readCh:  make(chan []byte, 32),
	}
	go mc.clientWriter()
	go mc.clientReader()
	return mc
}

// ClientWriteBytes sets next read operation content.
func (mc *MockConn) ClientWriteBytes(b []byte) {
	mc.writeCh <- b
}

// ClientWriteElement sets next read operation content from
// a serialized XML element.
func (mc *MockConn) ClientWriteElement(elem xml.XElement) {
	buf := new(bytes.Buffer)
	elem.ToXML(buf, true)
	mc.ClientWriteBytes(buf.Bytes())
}

// ClientReadBytes retrieves previous write operation written bytes.
func (mc *MockConn) ClientReadBytes() []byte {
	select {
	case b := <-mc.readCh:
		return b
	case <-time.After(time.Second * 3):
		return nil
	}
}

// ClientReadElement deserializes previous write operation content
// into an XML elements array.
func (mc *MockConn) ClientReadElement() xml.XElement {
	if mc.r != nil && mc.r.Len() > 0 {
		goto doRead
	}
	mc.r = bytes.NewReader(<-mc.readCh)
	mc.p = xml.NewParser(mc.r)
doRead:
	el, _ := mc.p.ParseElement()
	if el == nil {
		goto doRead
	}
	return el
}

// Read performs a read operation on the mocked connection.
func (mc *MockConn) Read(b []byte) (n int, err error) {
	return mc.srvPipe.r.Read(b)
}

// Write performs a write operation on the mocked connection.
func (mc *MockConn) Write(b []byte) (n int, err error) {
	return mc.clPipe.w.Write(b)
}

// Close marks mocked connection as closed.
func (mc *MockConn) Close() error {
	atomic.StoreUint32(&mc.closed, 1)
	mc.clPipe.r.Close()
	close(mc.writeCh)
	mc.discCh <- true
	return nil
}

// WaitClose expects until the mocked connection closes.
func (mc *MockConn) WaitClose() bool {
	return mc.WaitCloseWithTimeout(time.Second)
}

// WaitCloseWithTimeout expects until the mocked connection closes
// or until a timeout fires.
func (mc *MockConn) WaitCloseWithTimeout(timeout time.Duration) bool {
	select {
	case <-mc.discCh:
		return true
	case <-time.After(timeout):
		return false
	}
}

// IsClosed returns whether or not the mocked connection
// has been closed.
func (mc *MockConn) IsClosed() bool {
	return atomic.LoadUint32(&mc.closed) == 1
}

// LocalAddr returns a mocked remote address.
func (mc *MockConn) LocalAddr() net.Addr {
	return &mockConnAddress{
		network: mockConnNetwork,
		str:     mockConnLocalAddr,
	}
}

// RemoteAddr returns a mocked remote address.
func (mc *MockConn) RemoteAddr() net.Addr {
	return &mockConnAddress{
		network: mockConnNetwork,
		str:     mockConnRemoteAddr,
	}
}

// SetDeadline satisfies net.Conn interface.
func (mc *MockConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline satisfies net.Conn interface.
func (mc *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline satisfies net.Conn interface.
func (mc *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (mc *MockConn) clientWriter() {
	for b := range mc.writeCh {
		mc.srvPipe.w.Write(b)
	}
}

func (mc *MockConn) clientReader() {
	for {
		bt := make([]byte, 8192)
		n, err := mc.clPipe.r.Read(bt)
		switch err {
		case nil:
			break
		case io.EOF:
			if n > 0 {
				mc.readCh <- bt[:n]
				return
			}
		default:
			return
		}
		mc.readCh <- bt[:n]
	}
}
