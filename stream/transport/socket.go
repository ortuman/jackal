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

	"github.com/ortuman/jackal/config"
)

type socketTransport struct {
	conn         net.Conn
	br           *bufio.Reader
	bw           *bufio.Writer
	readDeadline time.Duration
}

func NewSocketTransport(conn net.Conn, keepAlive int) Transport {
	s := &socketTransport{
		conn:         conn,
		br:           bufio.NewReader(conn),
		bw:           bufio.NewWriter(conn),
		readDeadline: time.Second * time.Duration(keepAlive),
	}
	return s
}

func (s *socketTransport) Write(p []byte) (n int, err error) {
	defer s.bw.Flush()
	return s.bw.Write(p)
}

func (s *socketTransport) Read(p []byte) (n int, err error) {
	s.conn.SetReadDeadline(time.Now().Add(s.readDeadline))
	return s.br.Read(p)
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
}

func (s *socketTransport) ChannelBindingBytes(mechanism string) []byte {
	if tlsConn, ok := s.conn.(*tls.Conn); ok {
		switch mechanism {
		case TLSUnique:
			st := tlsConn.ConnectionState()
			return st.TLSUnique
		default:
			break
		}
	}
	return []byte{}
}
