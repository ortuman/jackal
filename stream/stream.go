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
	tr *transport.Transport
}

func NewStreamSocket(conn net.Conn, maxReadCount, keepAlive int) *Stream {
	s := &Stream{}
	s.tr = transport.NewSocketTransport(conn, s, maxReadCount, keepAlive)
	return s
}

func (s *Stream) ReadBytes(b []byte) {
	l := strings.TrimSpace(string(b))
	if l == "quit" {
		s.tr.Close()
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
