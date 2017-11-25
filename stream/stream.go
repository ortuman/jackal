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

	"github.com/ortuman/jackal/config"
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

const (
	InvalidXMLStreamError            = "invalid-xml"
	InvalidNamespaceStreamError      = "invalid-namespace"
	HostUnknownStreamError           = "host-unknown"
	InvalidFromStreamError           = "invalid-from"
	ConnectionTimeoutStreamError     = "connection-timeout"
	UnsupportedStanzaTypeStreamError = "unsupported-stanza-type"
	UnsupportedVersionStreamError    = "unsupported-version"
	NotAuthorizedStreamError         = "not-authorized"
	InternalServerErrorStreamError   = "internal-server-error"
)

type Stream struct {
	sync.RWMutex
	cfg           config.Server
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

	writeCh chan []byte
	readCh  chan []byte
	discCh  chan string
}

func NewStreamSocket(id string, conn net.Conn, maxReadCount, keepAlive int, config config.Server) *Stream {
	s := &Stream{
		cfg:     config,
		id:      id,
		parser:  xml.NewParser(),
		st:      connecting,
		writeCh: make(chan []byte, 32),
		readCh:  make(chan []byte, 32),
		discCh:  make(chan string, 1),
	}
	s.tr = transport.NewSocketTransport(conn, s, maxReadCount, keepAlive)
	go s.loop()
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

func (s *Stream) state() int32 {
	return atomic.LoadInt32(&s.st)
}

func (s *Stream) setState(state int32) {
	atomic.StoreInt32(&s.st, state)
}

func (s *Stream) SendElements(elems []*xml.Element) {
	for _, e := range elems {
		s.SendElement(e)
	}
}

func (s *Stream) SendElement(elem *xml.Element) {
	s.writeCh <- []byte(elem.XML(true))
}

func (s *Stream) ReadBytes(b []byte) {
	s.readCh <- b
}

func (s *Stream) Error(err error) {
	log.Errorf("%v", err)
}

func (s *Stream) handleElement(elem *xml.Element) {
	switch s.state() {
	case connecting:
		s.handleConnecting(elem)
	default:
		break
	}
}

func (s *Stream) handleConnecting(elem *xml.Element) {
	// if err := s.validateStreamElement(); err != nil {
	// 	return
	// }
}

func (s *Stream) loop() {
	for {
		// stop looping after disconnecting stream
		if s.state() == disconnected {
			return
		}
		select {
		case b := <-s.writeCh:
			s.tr.Write(b)

		case b := <-s.readCh:
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
				s.disconnectWithStreamError(InvalidXMLStreamError)
			}

		case strmErr := <-s.discCh:
			s.disconnectWithStreamError(strmErr)
		}
	}
}

func (s *Stream) streamDefaultNamespace() string {
	return ""
}

func (s *Stream) disconnectWithStreamError(strmErr string) {
	if s.state() == connected {
		s.disconnect(false)
	} else {
		s.disconnect(true)
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

	Manager().UnregisterStream(s)
}

func streamErrorElement(strmErr string) *xml.Element {
	ret := xml.NewMutableElementName("stream:error")
	reason := xml.NewElementNamespace(strmErr, "urn:ietf:params:xml:ns:xmpp-streams")
	ret.AppendElement(reason)
	return ret.Copy()
}
