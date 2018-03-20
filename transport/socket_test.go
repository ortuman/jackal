// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package transport

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ortuman/jackal/transport/compress"
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
func (c *fakeSocketConn) SetDeadline(_ time.Time) error      { return nil }
func (c *fakeSocketConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *fakeSocketConn) SetWriteDeadline(_ time.Time) error { return nil }

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
	st := NewSocketTransport(conn)
	st2 := st.(*socketTransport)

	str := `<elem xmlns="exodus:ns"/>`
	_, _ = io.WriteString(st, str)
	_ = st.Flush()
	require.Equal(t, str, string(conn.w.Bytes()))

	str2 := `<elem2 xmlns="exodus2:ns"/>`
	_, _ = io.WriteString(conn.r, str2)

	n, err := st.Read(buff)
	require.Nil(t, err)
	require.Equal(t, str2, string(buff[:n]))

	st.EnableCompression(compress.BestCompression)
	require.True(t, st2.compressed)

	st.(*socketTransport).conn = &net.TCPConn{}
	st.StartTLS(&tls.Config{}, false)
	_, ok := st2.conn.(*tls.Conn)
	require.True(t, ok)
	st.(*socketTransport).conn = conn

	require.Nil(t, st2.ChannelBindingBytes(ChannelBindingMechanism(99)))
	require.Nil(t, st2.ChannelBindingBytes(TLSUnique))

	_ = st.Close()
	require.True(t, conn.closed)
}
