/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package transport

import (
	"io"
	"net"
)

type writeReq struct {
	r          io.Reader
	continueCh chan struct{}
}

type enableCompressionReq struct {
	level      int
	continueCh chan struct{}
}

type socketTransport struct {
	conn      net.Conn
	keepAlive int

	writeCh             chan writeReq
	startTLSCh          chan chan struct{}
	enableCompressionCh chan enableCompressionReq
	cbBytesCh           chan chan []byte
	closeCh             chan chan struct{}
}

func NewSocketTransport(conn net.Conn, keepAlive int) *Transport {
	s := &socketTransport{
		conn:                conn,
		keepAlive:           keepAlive,
		writeCh:             make(chan writeReq),
		startTLSCh:          make(chan chan struct{}),
		enableCompressionCh: make(chan enableCompressionReq),
		cbBytesCh:           make(chan chan []byte),
		closeCh:             make(chan chan struct{}),
	}
	go s.writeLoop()

	return &Transport{
		Write:               s.Write,
		WriteAndWait:        s.WriteAndWait,
		Close:               s.Close,
		StartTLS:            s.StartTLS,
		EnableCompression:   s.EnableCompression,
		ChannelBindingBytes: s.ChannelBindingBytes,
	}
}

func (s *socketTransport) Write(b io.Reader) {
	req := writeReq{
		r:          b,
		continueCh: make(chan struct{}),
	}
	s.writeCh <- req
}

func (s *socketTransport) WriteAndWait(b io.Reader) {
	continueCh := make(chan struct{})
	req := writeReq{
		r:          b,
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
		case <-s.closeCh:
			alive = false
		}
	}
}
