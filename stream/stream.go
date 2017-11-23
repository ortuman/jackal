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
	return s
}
