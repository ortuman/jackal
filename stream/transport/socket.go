/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

import (
	"bufio"
	"crypto/tls"
	"net"
	"time"

	"io"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/stream/compress"
	"github.com/ortuman/jackal/stream/compress/zlib"
)

const socketBufferSize = 8192

type socketTransport struct {
	conn         net.Conn
	w            io.Writer
	r            io.Reader
	br           *bufio.Reader
	bw           *bufio.Writer
	readDeadline time.Duration
	compressor   compress.Compressor
}

func NewSocketTransport(conn net.Conn, keepAlive int) Transport {
	s := &socketTransport{
		conn:         conn,
		br:           bufio.NewReaderSize(conn, socketBufferSize),
		bw:           bufio.NewWriterSize(conn, socketBufferSize),
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
	if s.compressor == nil {
		s.compressor = zlib.NewCompressor(s.br, s.bw, level)
		s.w = s.compressor
		s.r = s.compressor
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
	return []byte{}
}
