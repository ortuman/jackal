/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"bytes"
	"net"
	"sync"
	"sync/atomic"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream/transport"
	"github.com/ortuman/jackal/xml"
)

const (
	connecting = iota
	connected
	authenticating
	authenticated
	sessionStarted
	disconnected
)

type Stream struct {
	sync.RWMutex
	tr            *transport.Transport
	parser        *xml.Parser
	st            int32
	id            string
	username      string
	domain        string
	resource      string
	secured       bool
	authenticated bool
	compressed    bool

	procReadCh chan []byte
	closeCh    chan struct{}
}

func NewStreamSocket(id string, conn net.Conn, maxReadCount, keepAlive int) *Stream {
	s := &Stream{
		id:         id,
		parser:     xml.NewParser(),
		st:         connecting,
		procReadCh: make(chan []byte, 1),
		closeCh:    make(chan struct{}),
	}
	s.tr = transport.NewSocketTransport(conn, s, maxReadCount, keepAlive)
	go s.procElementLoop()
	return s
}

func (s *Stream) ID() string {
	return s.id
}

func (s *Stream) Username() string {
	s.RLock()
	defer s.RUnlock()
	return s.username
}

func (s *Stream) Domain() string {
	s.RLock()
	defer s.RUnlock()
	return s.domain
}

func (s *Stream) Resource() string {
	s.RLock()
	defer s.RUnlock()
	return s.resource
}

func (s *Stream) SendElements(elems []*xml.Element) {
	for _, e := range elems {
		s.SendElement(e)
	}
}

func (s *Stream) SendElement(elem *xml.Element) {
	s.tr.Write([]byte(elem.XML(true)))
}

func (s *Stream) ReadBytes(b []byte) {
	s.procReadCh <- b
}

func (s *Stream) Error(err error) {
	log.Errorf("%v", err)
}

func (s *Stream) handleElement(e *xml.Element) {
}

func (s *Stream) procElementLoop() {
	for {
		// stop processing reads after disconnecting stream
		if s.state() == disconnected {
			return
		}
		select {
		case b := <-s.procReadCh:
			// stream closed by client
			if "</stream:stream>" == string(b) {
				s.disconnect(false)
				continue
			}
			if err := s.parser.ParseElements(bytes.NewReader(b)); err == nil {
				e := s.parser.PopElement()
				for e != nil {
					s.handleElement(e)
					e = s.parser.PopElement()
				}
			} else { // XML parsing error
				log.Errorf("%v", err)
				s.disconnect(false)
			}
		}
	}
}

func (s *Stream) disconnect(closeStream bool) {
	if closeStream {
		s.tr.WriteAndWait([]byte("</stream:stream>"))
	}
	s.tr.Close()

	s.Lock()
	s.authenticated = false
	s.secured = false
	s.compressed = false
	s.Unlock()

	s.setState(disconnected)
}

func (s *Stream) state() int32 {
	return atomic.LoadInt32(&s.st)
}

func (s *Stream) setState(state int32) {
	atomic.StoreInt32(&s.st, state)
}
