/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"bytes"
	"errors"
	"net"
	"sync"
	"sync/atomic"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream/transport"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

const (
	connecting = iota
	connected
	authenticating
	authenticated
	sessionStarted
	disconnected
)

var InvalidXMLStreamError = errors.New("invalid-xml")
var InvalidNamespaceStreamError = errors.New("invalid-namespace")
var HostUnknownStreamError = errors.New("host-unknown")
var InvalidFromStreamError = errors.New("invalid-from")
var ConnectionTimeoutStreamError = errors.New("connection-timeout")
var UnsupportedStanzaTypeStreamError = errors.New("unsupported-stanza-type")
var UnsupportedVersionStreamError = errors.New("unsupported-version")
var NotAuthorizedStreamError = errors.New("not-authorized")
var InternalServerErrorStreamError = errors.New("internal-server-error")

type Stream struct {
	sync.RWMutex
	cfg           *config.Server
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

	writeCh chan *xml.Element
	readCh  chan []byte
	discCh  chan error
}

func NewStreamSocket(id string, conn net.Conn, config *config.Server) *Stream {
	s := &Stream{
		cfg:     config,
		id:      id,
		parser:  xml.NewParser(),
		st:      connecting,
		writeCh: make(chan *xml.Element, 32),
		readCh:  make(chan []byte, 32),
		discCh:  make(chan error, 1),
	}
	// assign default domain
	s.domain = s.cfg.Domains[0]

	maxReadCount := config.Transport.MaxStanzaSize
	keepAlive := config.Transport.KeepAlive
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

func (s *Stream) SendElements(elems []*xml.Element) {
	for _, e := range elems {
		s.SendElement(e)
	}
}

func (s *Stream) SendElement(elem *xml.Element) {
	s.writeCh <- elem
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
		case elem := <-s.writeCh:
			s.writeElement(elem)

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

func (s *Stream) openStreamElement() {
	ops := xml.NewMutableElementName("stream:stream")
	ops.SetAttribute("xmlns", s.streamDefaultNamespace())
	ops.SetAttribute("xmlns:stream", "http://etherx.jabber.org/streams")
	ops.SetAttribute("id", uuid.New())
	ops.SetAttribute("from", s.Domain())
	ops.SetAttribute("version", "1.0")

	s.tr.WriteAndWait([]byte(`<?xml version="1.0"?>`))
	s.tr.WriteAndWait([]byte(ops.XML(false)))
}

func (s *Stream) streamDefaultNamespace() string {
	switch s.cfg.Type {
	case config.C2S:
		return "jabber:client"
	case config.S2S:
		return "jabber:server"
	default:
		// should not be reached
		log.Fatalf("unrecognized server type: %s", s.cfg.Type)
		return ""
	}
}

func (s *Stream) writeElement(elem *xml.Element) {
	b := []byte(elem.XML(true))
	s.tr.Write(b)
}

func (s *Stream) writeElementAndWait(elem *xml.Element) {
	b := []byte(elem.XML(true))
	s.tr.WriteAndWait(b)
}

func (s *Stream) disconnectWithStreamError(err error) {
	if s.state() == connecting {
		s.openStreamElement()
	}
	strmErr := streamErrorElement(err)
	s.writeElementAndWait(strmErr)
	s.disconnect(true)
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

func (s *Stream) state() int32 {
	return atomic.LoadInt32(&s.st)
}

func (s *Stream) setState(state int32) {
	atomic.StoreInt32(&s.st, state)
}

func streamErrorElement(err error) *xml.Element {
	ret := xml.NewMutableElementName("stream:error")
	reason := xml.NewElementNamespace(err.Error(), "urn:ietf:params:xml:ns:xmpp-streams")
	ret.AppendElement(reason)
	return ret.Copy()
}
