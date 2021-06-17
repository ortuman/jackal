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
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ortuman/jackal/pkg/transport/compress"
	"github.com/ortuman/jackal/pkg/util/ratelimiter"
	"golang.org/x/time/rate"
)

const readBufferSize = 2048

var errNoWriteFlush = errors.New("transport: flushing buffer before writing")

var bufWriterPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewWriter(nil)
	},
}

type socketTransport struct {
	conn       net.Conn
	lr         *ratelimiter.Reader
	rd         io.Reader
	wr         io.Writer
	bw         *bufio.Writer
	compressed bool
}

// NewSocketTransport creates a socket class stream transport.
func NewSocketTransport(conn net.Conn) Transport {
	lr := ratelimiter.NewReader(conn)
	s := &socketTransport{
		conn: conn,
		lr:   lr,
		rd:   bufio.NewReaderSize(lr, readBufferSize),
		wr:   conn,
	}
	return s
}

func (s *socketTransport) Read(p []byte) (n int, err error) {
	return s.rd.Read(p)
}

func (s *socketTransport) Write(p []byte) (n int, err error) {
	if s.bw == nil {
		s.grabBuffWriter()
	}
	return s.bw.Write(p)
}

func (s *socketTransport) WriteString(str string) (int, error) {
	if s.bw == nil {
		s.grabBuffWriter()
	}
	n, err := io.Copy(s.bw, strings.NewReader(str))
	return int(n), err
}

func (s *socketTransport) Close() error {
	return s.conn.Close()
}

func (s *socketTransport) Type() Type {
	return Socket
}

func (s *socketTransport) Flush() error {
	if s.bw == nil {
		return errNoWriteFlush
	}
	if err := s.bw.Flush(); err != nil {
		return err
	}
	s.releaseBuffWriter()
	return nil
}

func (s *socketTransport) SetWriteDeadline(d time.Time) error {
	return s.conn.SetWriteDeadline(d)
}

func (s *socketTransport) SetReadRateLimiter(rLim *rate.Limiter) error {
	s.lr.SetReadRateLimiter(rLim)
	return nil
}

func (s *socketTransport) StartTLS(cfg *tls.Config, asClient bool) {
	_, ok := s.conn.(*net.TCPConn)
	if !ok {
		return
	}
	if asClient {
		s.conn = tls.Client(s.conn, cfg)
	} else {
		s.conn = tls.Server(s.conn, cfg)
	}
	lr := ratelimiter.NewReader(s.conn)
	if rLim := s.lr.ReadRateLimiter(); rLim != nil {
		lr.SetReadRateLimiter(rLim)
	}
	s.lr = lr
	s.rd = bufio.NewReaderSize(lr, readBufferSize)
	s.wr = s.conn
}

func (s *socketTransport) EnableCompression(level compress.Level) {
	if s.compressed {
		return
	}
	rw := compress.NewZlibCompressor(s.rd, s.wr, level)
	s.rd = rw
	s.wr = rw
	s.compressed = true
}

func (s *socketTransport) SupportsChannelBinding() bool {
	conn, ok := s.conn.(tlsStateQueryable)
	if !ok {
		return false
	}
	return conn.ConnectionState().Version < tls.VersionTLS13
}

func (s *socketTransport) ChannelBindingBytes(mechanism ChannelBindingMechanism) []byte {
	conn, ok := s.conn.(tlsStateQueryable)
	if !ok {
		return nil
	}
	switch mechanism {
	case TLSUnique:
		connSt := conn.ConnectionState()
		return connSt.TLSUnique
	default:
		break
	}
	return nil
}

func (s *socketTransport) PeerCertificates() []*x509.Certificate {
	conn, ok := s.conn.(tlsStateQueryable)
	if !ok {
		return nil
	}
	st := conn.ConnectionState()
	return st.PeerCertificates
}

func (s *socketTransport) grabBuffWriter() {
	if s.bw != nil {
		return
	}
	s.bw = bufWriterPool.Get().(*bufio.Writer)
	s.bw.Reset(s.wr)
}

func (s *socketTransport) releaseBuffWriter() {
	if s.bw == nil {
		return
	}
	bufWriterPool.Put(s.bw)
	s.bw = nil
}
