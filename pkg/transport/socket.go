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
	"io"
	"net"
	"strings"
	"time"

	"github.com/ortuman/jackal/pkg/transport/compress"
	"github.com/ortuman/jackal/pkg/util/ratelimiter"
	"golang.org/x/time/rate"
)

const writeBuffSize = 4096

type readWriter struct {
	io.Reader
	io.Writer
}

type socketTransport struct {
	conn       net.Conn
	lr         *ratelimiter.Reader
	bw         *bufio.Writer
	rw         io.ReadWriter
	compressed bool
}

// NewSocketTransport creates a socket class stream transport.
func NewSocketTransport(conn net.Conn) Transport {
	lr := ratelimiter.NewReader(conn)
	bw := bufio.NewWriterSize(conn, writeBuffSize)
	s := &socketTransport{
		conn: conn,
		lr:   lr,
		bw:   bw,
		rw:   &readWriter{lr, bw},
	}
	return s
}

func (s *socketTransport) Read(p []byte) (n int, err error) {
	return s.rw.Read(p)
}

func (s *socketTransport) Write(p []byte) (n int, err error) {
	return s.rw.Write(p)
}

func (s *socketTransport) WriteString(str string) (int, error) {
	n, err := io.Copy(s.rw, strings.NewReader(str))
	return int(n), err
}

func (s *socketTransport) Close() error {
	return s.conn.Close()
}

func (s *socketTransport) Type() Type {
	return Socket
}

func (s *socketTransport) Flush() error {
	return s.bw.Flush()
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
	s.bw = bufio.NewWriterSize(s.conn, writeBuffSize)
	s.rw = &readWriter{s.lr, s.bw}
}

func (s *socketTransport) EnableCompression(level compress.Level) {
	if s.compressed {
		return
	}
	s.rw = compress.NewZlibCompressor(s.rw, s.rw, level)
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
