/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"time"

	"github.com/ortuman/jackal/transport/compress"
	"github.com/ortuman/jackal/xml"
)

const socketBuffSize = 4096

type socketTransport struct {
	conn       net.Conn
	rw         io.ReadWriter
	br         *bufio.Reader
	bw         *bufio.Writer
	keepAlive  time.Duration
	compressed bool
}

// NewSocketTransport creates a socket class stream transport.
func NewSocketTransport(conn net.Conn, keepAlive time.Duration) Transport {
	s := &socketTransport{
		conn:      conn,
		rw:        conn,
		br:        bufio.NewReaderSize(conn, socketBuffSize),
		bw:        bufio.NewWriterSize(conn, socketBuffSize),
		keepAlive: keepAlive,
	}
	return s
}

func (s *socketTransport) Type() TransportType {
	return Socket
}

func (s *socketTransport) Read(p []byte) (n int, err error) {
	s.conn.SetReadDeadline(time.Now().Add(s.keepAlive))
	return s.br.Read(p)
}

func (s *socketTransport) Close() error {
	return s.conn.Close()
}

func (s *socketTransport) WriteString(str string) error {
	defer s.bw.Flush()
	_, err := io.WriteString(s.bw, str)
	return err
}

func (s *socketTransport) WriteElement(elem xml.XElement, includeClosing bool) error {
	defer s.bw.Flush()
	elem.ToXML(s.bw, includeClosing)
	return nil
}

func (s *socketTransport) StartTLS(cfg *tls.Config) {
	if _, ok := s.conn.(*tls.Conn); !ok {
		s.conn = tls.Server(s.conn, cfg)
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
	if tlsConn, ok := s.conn.(*tls.Conn); ok {
		switch mechanism {
		case TLSUnique:
			st := tlsConn.ConnectionState()
			return st.TLSUnique
		default:
			break
		}
	}
	return nil
}
