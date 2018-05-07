/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

type fakeWebSocketReader struct {
	buf *bytes.Buffer
}

func (r *fakeWebSocketReader) Read(p []byte) (n int, err error) { return r.buf.Read(p) }

type fakeWebSocketWriter struct {
	buf *bytes.Buffer
}

func (w *fakeWebSocketWriter) Write(p []byte) (n int, err error) { return w.buf.Write(p) }
func (w *fakeWebSocketWriter) Close() error                      { return nil }

type fakeWebSocketConn struct {
	r      *fakeWebSocketReader
	w      *fakeWebSocketWriter
	closed bool
}

func newFakeWebSocketConn() *fakeWebSocketConn {
	return &fakeWebSocketConn{
		r: &fakeWebSocketReader{buf: new(bytes.Buffer)},
		w: &fakeWebSocketWriter{buf: new(bytes.Buffer)},
	}
}

func (c *fakeWebSocketConn) NextReader() (messageType int, r io.Reader, err error) { return 0, c.r, nil }
func (c *fakeWebSocketConn) NextWriter(int) (writer io.WriteCloser, err error)     { return c.w, nil }
func (c *fakeWebSocketConn) Close() error                                          { c.closed = true; return nil }
func (c *fakeWebSocketConn) SetReadDeadline(t time.Time) error                     { return nil }
func (c *fakeWebSocketConn) UnderlyingConn() net.Conn                              { return &tls.Conn{} }

func TestWebSocketTransport(t *testing.T) {
	conn := newFakeWebSocketConn()

	// test read...
	iq := xml.NewIQType(uuid.New(), xml.ResultType)
	iq.SetFrom("localhost")
	iq.ToXML(conn.r.buf, true)

	wst := NewWebSocketTransport(conn, 16384, 10)
	el, err := wst.ReadElement()
	require.Nil(t, err)
	require.Equal(t, iq.String(), el.String())

	// test write...
	msg := xml.NewMessageType(uuid.New(), xml.ChatType)
	b := xml.NewElementName("body")
	b.SetText("Hi buddy!")
	msg.AppendElement(b)

	wst.WriteString(msg.String())
	require.Equal(t, msg.String(), conn.w.buf.String())
	conn.w.buf.Reset()

	wst.WriteElement(msg, true)
	require.Equal(t, msg.String(), conn.w.buf.String())

	require.Nil(t, wst.ChannelBindingBytes(ChannelBindingMechanism(99)))
	require.Nil(t, wst.ChannelBindingBytes(TLSUnique))

	wst.Close()
	require.True(t, conn.closed)
}
