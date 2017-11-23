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

type writeReq struct {
	b          []byte
	continueCh chan struct{}
}

type enableCompressionReq struct {
	level      int
	continueCh chan struct{}
}

type socketTransport struct {
	callback     Callback
	conn         net.Conn
	maxReadCount int
	keepAlive    int
	isClosed     int32

	writeCh             chan writeReq
	startTLSCh          chan chan struct{}
	enableCompressionCh chan enableCompressionReq
	cbBytesCh           chan chan []byte
	closeCh             chan chan struct{}
}

func NewSocketTransport(conn net.Conn, callback Callback, maxReadCount, keepAlive int) *Transport {
	s := &socketTransport{
		conn:                conn,
		callback:            callback,
		maxReadCount:        maxReadCount,
		keepAlive:           keepAlive,
		writeCh:             make(chan writeReq),
		startTLSCh:          make(chan chan struct{}),
		enableCompressionCh: make(chan enableCompressionReq),
		cbBytesCh:           make(chan chan []byte),
		closeCh:             make(chan chan struct{}),
	}
	go s.readLoop()
	go s.writeLoop()

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
	req := writeReq{
		b:          b,
		continueCh: make(chan struct{}),
	}
	s.writeCh <- req
}

func (s *socketTransport) WriteAndWait(b []byte) {
	continueCh := make(chan struct{})
	req := writeReq{
		b:          b,
		continueCh: continueCh,
	}
	s.writeCh <- req
	<-continueCh
}

func (s *socketTransport) Close() {
	continueCh := make(chan struct{})
	s.closeCh <- continueCh
	<-continueCh
}

func (s *socketTransport) StartTLS() {
	continueCh := make(chan struct{})
	s.startTLSCh <- continueCh
	<-continueCh
}

func (s *socketTransport) EnableCompression(level int) {
	continueCh := make(chan struct{})
	req := enableCompressionReq{
		level:      level,
		continueCh: continueCh,
	}
	s.enableCompressionCh <- req
	<-continueCh
}

func (s *socketTransport) ChannelBindingBytes(mechanism int) []byte {
	bytesCh := make(chan []byte)
	s.cbBytesCh <- bytesCh
	return <-bytesCh
}

func (s *socketTransport) writeLoop() {
	alive := true
	for alive {
		select {
		case req := <-s.writeCh:
			s.conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			_, err := s.conn.Write(req.b)
			if err == nil {
				s.callback.SentBytes(req.b)
			} else {
				s.callback.Error(err)
			}
			close(req.continueCh)

		case continueCh := <-s.startTLSCh:
			close(continueCh)

		case req := <-s.enableCompressionCh:
			close(req.continueCh)

		case respCh := <-s.cbBytesCh:
			respCh <- []byte{}

		case continueCh := <-s.closeCh:
			alive = false
			atomic.StoreInt32(&s.isClosed, 1)
			s.conn.Close()
			close(continueCh)
		}
	}
}

func (s *socketTransport) readLoop() {
	buff := make([]byte, s.maxReadCount)
	for {
		n, err := s.conn.Read(buff)
		if atomic.LoadInt32(&s.isClosed) == 1 {
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
