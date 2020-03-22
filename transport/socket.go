/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"strings"
	"time"

	"github.com/ortuman/jackal/transport/compress"
)

const socketBuffSize = 4096

type socketTransport struct {
	conn       net.Conn
	rw         io.ReadWriter
	br         *bufio.Reader
	bw         *bufio.Writer
	compressed bool
}

// NewSocketTransport creates a socket class stream transport.
func NewSocketTransport(conn net.Conn) Transport {
	s := &socketTransport{
		conn: conn,
		rw:   conn,
		br:   bufio.NewReaderSize(conn, socketBuffSize),
		bw:   bufio.NewWriterSize(conn, socketBuffSize),
	}
	return s
}

func (s *socketTransport) Read(p []byte) (n int, err error) {
	return s.br.Read(p)
}

func (s *socketTransport) Write(p []byte) (n int, err error) {
	return s.bw.Write(p)
}

func (s *socketTransport) Close() error {
	return s.conn.Close()
}

func (s *socketTransport) Type() Type {
	return Socket
}

func (s *socketTransport) WriteString(str string) (int, error) {
	n, err := io.Copy(s.bw, strings.NewReader(str))
	return int(n), err
}

// Flush writes any buffered data to the underlying io.Writer.
func (s *socketTransport) Flush() error {
	return s.bw.Flush()
}

// SetWriteDeadline sets the deadline for future write calls.
func (s *socketTransport) SetWriteDeadline(d time.Time) error {
	return s.conn.SetWriteDeadline(d)
}

func (s *socketTransport) StartTLS(cfg *tls.Config, asClient bool) {
	if _, ok := s.conn.(*net.TCPConn); ok {
		if asClient {
			s.conn = tls.Client(s.conn, cfg)
		} else {
			s.conn = tls.Server(s.conn, cfg)
		}
		s.rw = s.conn
		s.bw.Reset(s.rw)
		s.br.Reset(s.rw)
	}
}

func (s *socketTransport) EnableCompression(level compress.Level) {
	if !s.compressed {
		s.rw = compress.NewZlibCompressor(s.rw, s.rw, level)
		s.bw.Reset(s.rw)
		s.br.Reset(s.rw)
		s.compressed = true
	}
}

func (s *socketTransport) ChannelBindingBytes(mechanism ChannelBindingMechanism) []byte {
	if conn, ok := s.conn.(tlsStateQueryable); ok {
		switch mechanism {
		case TLSUnique:
			st := conn.ConnectionState()
			return st.TLSUnique
		default:
			break
		}
	}
	return nil
}

func (s *socketTransport) PeerCertificates() []*x509.Certificate {
	if conn, ok := s.conn.(tlsStateQueryable); ok {
		st := conn.ConnectionState()
		return st.PeerCertificates
	}
	return nil
}
