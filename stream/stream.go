/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import "github.com/ortuman/jackal/server/transport"

type Stream struct {
	tr *transport.Transport
}

func New(transport *transport.Transport) *Stream {
	s := &Stream{
		tr: transport,
	}
	transport.Callback = s
	return s
}

func (s *Stream) ReadBytes([]byte) {
}

func (s *Stream) SentBytes(int) {
}

func (s *Stream) StartedTLS() {
}

func (s *Stream) FailedStartTLS(error) {
}

func (s *Stream) Error(error) {
}
