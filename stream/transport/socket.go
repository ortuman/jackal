/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ortuman/jackal/stream/compress"
	"github.com/ortuman/jackal/stream/compress/zlib"
)

type socketTransport struct {
	sync.RWMutex
	conn       net.Conn
	keepAlive  int
	closed     bool
	readBuff   []byte
	compressor compress.Compressor
}

func NewSocketTransport(conn net.Conn, maxReadCount, keepAlive int) Transport {
	s := &socketTransport{
		conn:      conn,
		keepAlive: keepAlive,
		readBuff:  make([]byte, maxReadCount),
	}
	return s
}

func (s *socketTransport) Write(b []byte) {
	go s.writeBytes(b)
}

func (s *socketTransport) WriteAndWait(b []byte) {
	s.writeBytes(b)
}

func (s *socketTransport) Read() ([]byte, error) {
	s.RLock()
	defer s.RUnlock()

	readDeadline := time.Now().Add(time.Second * time.Duration(s.keepAlive))
	s.conn.SetReadDeadline(readDeadline)
	n, err := s.conn.Read(s.readBuff)
	if s.closed {
		return nil, ErrServerClosedTransport
	}
	switch err {
	case nil:
		b := s.readBuff[:n]
		if s.compressor != nil {
			return s.compressor.Uncompress(b)
		}
		return b, nil

	case io.EOF:
		return nil, ErrRemotePeerClosedTransport
	default:
		return nil, err
	}
}

func (s *socketTransport) Close() {
	s.Lock()
	defer s.Unlock()
	s.conn.Close()
	s.closed = true
}

func (s *socketTransport) StartTLS(cfg *tls.Config) {
	s.Lock()
	defer s.Unlock()
	s.conn = tls.Server(s.conn, cfg)
}

func (s *socketTransport) EnableCompression(level compress.Level) {
	s.Lock()
	defer s.Unlock()
	s.compressor = zlib.NewCompressor(level)
}

func (s *socketTransport) ChannelBindingBytes(mechanism string) []byte {
	s.RLock()
	defer s.RLock()

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

func (s *socketTransport) writeBytes(b []byte) {
	s.RLock()
	defer s.RUnlock()

	if s.compressor != nil {
		deflatedBytes, err := s.compressor.Compress(b)
		if deflatedBytes != nil && err == nil {
			s.conn.Write(deflatedBytes)
		}
	} else {
		s.conn.Write(b)
	}
}
