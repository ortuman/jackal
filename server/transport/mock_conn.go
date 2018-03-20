/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bytes"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/bufferpool"
	"github.com/ortuman/jackal/xml"
)

const mockConnNetwork = "tcp"

var pool = bufferpool.New()

const (
	mockConnLocalAddr  = "10.188.17.228"
	mockConnRemoteAddr = "77.230.105.223"
)

type readReq struct {
	p   []byte
	err error
}

type mockAddress struct {
	network string
	str     string
}

func (ma *mockAddress) Network() string {
	return ma.network
}

func (ma *mockAddress) String() string {
	return ma.str
}

// MockConn represents a net.Conn mocked implementation.
type MockConn struct {
	closed uint32
	wb     *bytes.Buffer
	wbMu   sync.Mutex
	readCh chan readReq
	discCh chan bool
}

// NewMockConn returns a new initialized MockConn instance.
func NewMockConn() *MockConn {
	return &MockConn{
		wb:     new(bytes.Buffer),
		readCh: make(chan readReq, 1),
		discCh: make(chan bool, 1),
	}
}

// Read performs a read operation on the mocked connection.
func (mc *MockConn) Read(b []byte) (n int, err error) {
	r := <-mc.readCh
	if len(r.p) > 0 {
		copy(b, r.p)
	}
	return len(r.p), r.err
}

// SendBytes sets next read operation content.
func (mc *MockConn) SendBytes(b []byte) {
	mc.readCh <- readReq{p: b, err: nil}
}

// SendElement sets next read operation content from
// a serialized XML element.
func (mc *MockConn) SendElement(elem xml.Element) {
	buf := pool.Get()
	defer pool.Put(buf)
	elem.ToXML(buf, true)
	mc.SendBytes(buf.Bytes())
}

// Write performs a write operation on the mocked connection.
func (mc *MockConn) Write(b []byte) (n int, err error) {
	mc.wbMu.Lock()
	mc.wb.Write(b)
	mc.wbMu.Unlock()
	return len(b), nil
}

// ReadBytes retrieves previous write operation written bytes.
func (mc *MockConn) ReadBytes() []byte {
	mc.wbMu.Lock()
	b := mc.wb.Bytes()
	mc.wb.Reset()
	mc.wbMu.Unlock()
	return b
}

// ReadElements deserializes previous write operation content
// into an XML elements array.
func (mc *MockConn) ReadElements() []xml.Element {
	p := xml.NewParser(bytes.NewBuffer(mc.ReadBytes()))
	var elems []xml.Element
	el, err := p.ParseElement()
	for err != io.EOF {
		elems = append(elems, el)
		el, err = p.ParseElement()
	}
	return elems
}

// Close marks mocked connection as closed.
func (mc *MockConn) Close() error {
	atomic.StoreUint32(&mc.closed, 1)
	mc.discCh <- true
	mc.readCh <- readReq{p: nil, err: io.EOF}
	return nil
}

// WaitClose expects until the mocked connection closes.
func (mc *MockConn) WaitClose() {
	<-mc.discCh
}

// WaitCloseWithTimeout expects until the mocked connection closes
// or until a timeout fires.
func (mc *MockConn) WaitCloseWithTimeout(timeout time.Duration) {
	select {
	case <-mc.discCh:
		break
	case <-time.After(timeout):
		break
	}
}

// IsClosed returns whether or not the mocked connection
// has been closed.
func (mc *MockConn) IsClosed() bool {
	return atomic.LoadUint32(&mc.closed) == 1
}

// LocalAddr returns a mocked remote address.
func (mc *MockConn) LocalAddr() net.Addr {
	return &mockAddress{
		network: mockConnNetwork,
		str:     mockConnLocalAddr,
	}
}

// RemoteAddr returns a mocked remote address.
func (mc *MockConn) RemoteAddr() net.Addr {
	return &mockAddress{
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
