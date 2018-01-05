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

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/stream/compress"
)

type socketTransport struct {
	conn               net.Conn
	w                  io.Writer
	r                  io.Reader
	br                 *bufio.Reader
	bw                 *bufio.Writer
	readDeadline       time.Duration
	compressionEnabled bool
}

func NewSocketTransport(conn net.Conn, bufferSize, keepAlive int) Transport {
	s := &socketTransport{
		conn:         conn,
		br:           bufio.NewReaderSize(conn, bufferSize),
		bw:           bufio.NewWriterSize(conn, bufferSize),
		readDeadline: time.Second * time.Duration(keepAlive),
	}
	s.w = s.bw
	s.r = s.br
	return s
}

func (s *socketTransport) Write(p []byte) (n int, err error) {
	defer s.bw.Flush()
	return s.w.Write(p)
}

func (s *socketTransport) Read(p []byte) (n int, err error) {
	s.conn.SetReadDeadline(time.Now().Add(s.readDeadline))
	return s.r.Read(p)
}

func (s *socketTransport) Close() error {
	return s.conn.Close()
}

func (s *socketTransport) StartTLS(cfg *tls.Config) {
	if _, ok := s.conn.(*tls.Conn); !ok {
		s.conn = tls.Server(s.conn, cfg)
		s.bw.Reset(s.conn)
		s.br.Reset(s.conn)
	}
}

func (s *socketTransport) EnableCompression(level config.CompressionLevel) {
	if !s.compressionEnabled {
		zwr := compress.NewZlibCompressor(s.br, s.bw, level)
		s.w = zwr
		s.r = zwr
		s.compressionEnabled = true
	}
}

func (s *socketTransport) ChannelBindingBytes(mechanism config.ChannelBindingMechanism) []byte {
	if tlsConn, ok := s.conn.(*tls.Conn); ok {
		switch mechanism {
		case config.TLSUnique:
			st := tlsConn.ConnectionState()
			return st.TLSUnique
		default:
			break
		}
	}
	return nil
}
