/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bytes"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/ortuman/jackal/transport/compress"
	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

type fakeSocketConn struct {
	r      *bytes.Buffer
	w      *bytes.Buffer
	closed bool
}

func newFakeSocketConn() *fakeSocketConn {
	return &fakeSocketConn{
		r: new(bytes.Buffer),
		w: new(bytes.Buffer),
	}
}

func (c *fakeSocketConn) Read(b []byte) (n int, err error)   { return c.r.Read(b) }
func (c *fakeSocketConn) Write(b []byte) (n int, err error)  { return c.w.Write(b) }
func (c *fakeSocketConn) Close() error                       { c.closed = true; return nil }
func (c *fakeSocketConn) LocalAddr() net.Addr                { return localAddr }
func (c *fakeSocketConn) RemoteAddr() net.Addr               { return remoteAddr }
func (c *fakeSocketConn) SetDeadline(t time.Time) error      { return nil }
func (c *fakeSocketConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *fakeSocketConn) SetWriteDeadline(t time.Time) error { return nil }

type fakeAddr int

var (
	localAddr  = fakeAddr(1)
	remoteAddr = fakeAddr(2)
)

func (a fakeAddr) Network() string { return "net" }
func (a fakeAddr) String() string  { return "str" }

func TestSocket(t *testing.T) {
	buff := make([]byte, 4096)
	conn := newFakeSocketConn()
	st := NewSocketTransport(conn, 4096)
	st2 := st.(*socketTransport)

	el1 := xml.NewElementNamespace("elem", "exodus:ns")
	el1.ToXML(st, true)
	require.Equal(t, 0, bytes.Compare([]byte(el1.String()), conn.w.Bytes()))

	el2 := xml.NewElementNamespace("elem2", "exodus2:ns")
	el2.ToXML(conn.r, true)
	n, err := st.Read(buff)
	require.Nil(t, err)
	require.Equal(t, el2.String(), string(buff[:n]))

	st.EnableCompression(compress.BestCompression)
	require.True(t, st2.compressed)

	st.StartTLS(&tls.Config{}, false)
	_, ok := st2.conn.(*tls.Conn)
	require.True(t, ok)

	require.Nil(t, st2.ChannelBindingBytes(ChannelBindingMechanism(99)))
	require.Nil(t, st2.ChannelBindingBytes(TLSUnique))

	st.Close()
	require.True(t, conn.closed)
}
