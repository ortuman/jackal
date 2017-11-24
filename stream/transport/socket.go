/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

import (
	"io"
	"net"
	"sync/atomic"
	"time"
)

const writeDeadline = 10 * time.Second // Time allowed to write a message to the peer.

type socketTransport struct {
	callback     Callback
	conn         net.Conn
	maxReadCount int
	keepAlive    int
	closed       int32
}

func NewSocketTransport(conn net.Conn, callback Callback, maxReadCount, keepAlive int) *Transport {
	s := &socketTransport{
		conn:         conn,
		callback:     callback,
		maxReadCount: maxReadCount,
		keepAlive:    keepAlive,
	}
	go s.readLoop()

	t := &Transport{
		Write:               s.Write,
		WriteAndWait:        s.WriteAndWait,
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

func (s *socketTransport) Close() {
	atomic.StoreInt32(&s.closed, 1)
	s.conn.Close()
}

func (s *socketTransport) StartTLS() error {
	return nil
}

func (s *socketTransport) EnableCompression(level CompressionLevel) {
}

func (s *socketTransport) ChannelBindingBytes(mechanism ChannelBindingMechanism) []byte {
	return []byte{}
}

func (s *socketTransport) writeBytes(b []byte) {
	s.conn.SetWriteDeadline(time.Now().Add(writeDeadline))
	_, err := s.conn.Write(b)
	if err != nil {
		s.callback.Error(err)
	}
}

func (s *socketTransport) readLoop() {
	buff := make([]byte, s.maxReadCount)
	for {
		n, err := s.conn.Read(buff)
		if atomic.LoadInt32(&s.closed) == 1 {
			return
		}
		switch err {
		case io.EOF:
			return
		case nil:
			if n > 0 {
				s.callback.ReadBytes(buff[:n])
			}
		default:
			s.callback.Error(err)
			return
		}
	}
}
