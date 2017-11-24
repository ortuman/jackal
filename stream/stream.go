/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"net"
	"strings"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream/transport"
)

type Stream struct {
	id string
	tr *transport.Transport

	username string
	domain   string
	resource string
}

func NewStreamSocket(id string, conn net.Conn, maxReadCount, keepAlive int) *Stream {
	s := &Stream{}
	s.id = id
	s.tr = transport.NewSocketTransport(conn, s, maxReadCount, keepAlive)
	return s
}

func (s *Stream) ID() string {
	return s.id
}

func (s *Stream) Username() string {
	return s.username
}

func (s *Stream) Domain() string {
	return s.domain
}

func (s *Stream) Resource() string {
	return s.resource
}

func (s *Stream) ReadBytes(b []byte) {
	l := strings.TrimSpace(string(b))
	if l == "quit" {
		s.tr.Close()
		Manager().UnregisterStream(s)
		return
	}
	log.Infof("%s", l)
}

func (s *Stream) SentBytes(b []byte) {
}

func (s *Stream) StartedTLS() {
}

func (s *Stream) FailedStartTLS(error) {
}

func (s *Stream) Error(err error) {
	log.Errorf("%v", err)
}
