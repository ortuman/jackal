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
)

const writeDeadline = 10 * time.Second // Time allowed to write a message to the peer.

type socketTransport struct {
	conn      net.Conn
	keepAlive int
	closed    int32
	readBuff  []byte
}

func NewSocketTransport(conn net.Conn, maxReadCount, keepAlive int) *Transport {
	s := &socketTransport{
		conn:      conn,
		keepAlive: keepAlive,
		readBuff:  make([]byte, maxReadCount),
	}

	t := &Transport{
		Write:               s.Write,
		WriteAndWait:        s.WriteAndWait,
		Read:                s.Read,
		Close:               s.Close,
		StartTLS:            s.StartTLS,
		EnableCompression:   s.EnableCompression,
		ChannelBindingBytes: s.ChannelBindingBytes,
	}
	return t
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
		return s.readBuff[:n], nil
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

func (s *socketTransport) StartTLS(cfg *tls.Config) error {
	s.conn = tls.Server(s.conn, cfg)
	return nil
}

func (s *socketTransport) EnableCompression(level CompressionLevel) {
}

func (s *socketTransport) ChannelBindingBytes(mechanism ChannelBindingMechanism) []byte {
	return []byte{}
}

func (s *socketTransport) writeBytes(b []byte) {
	s.conn.SetWriteDeadline(time.Now().Add(writeDeadline))
	s.conn.Write(b)
}
