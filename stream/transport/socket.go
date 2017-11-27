/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

import (
	"crypto/tls"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/stream/compress"
)

const writeDeadline = 10 * time.Second // time allowed to write a message to the peer.

type socketTransport struct {
	conn       net.Conn
	keepAlive  int
	closed     int32
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
	n, err := s.conn.Read(s.readBuff)
	if atomic.LoadInt32(&s.closed) == 1 {
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
	atomic.StoreInt32(&s.closed, 1)
	s.conn.Close()
}

func (s *socketTransport) StartTLS(cfg *tls.Config) {
	s.conn = tls.Server(s.conn, cfg)
}

func (s *socketTransport) EnableCompression(level compress.Level) {
	s.compressor = compress.NewZLIBCompressor(level)
}

func (s *socketTransport) ChannelBindingBytes(mechanism ChannelBindingMechanism) []byte {
	if tlsConn, ok := s.conn.(*tls.Conn); ok {
		switch mechanism {
		case TLSUnique:
			st := tlsConn.ConnectionState()
			return st.TLSUnique
		default:
			return []byte{}
		}
	}
	return []byte{}
}

func (s *socketTransport) writeBytes(b []byte) {
	s.conn.SetWriteDeadline(time.Now().Add(writeDeadline))
	if s.compressor != nil {
		deflatedBytes, err := s.compressor.Compress(b)
		if deflatedBytes != nil && err == nil {
			s.conn.Write(deflatedBytes)
		}
	} else {
		s.conn.Write(b)
	}
}
