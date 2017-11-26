/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"bytes"
	"io"
	"net"
	"strings"
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

const streamNamespace = "http://etherx.jabber.org/streams"

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

	authenticators []authenticator

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

	// initialize authenticators
	s.initializeAuthenticators()

	// define transport callback
	cb := &transport.Callback{
		ReadBytes: func(b []byte) {
			s.readCh <- b
		},
		Close: func() {
			s.discCh <- io.EOF
		},
		Error: func(err error) {
			s.discCh <- err
		},
	}

	maxReadCount := config.Transport.MaxStanzaSize
	keepAlive := config.Transport.KeepAlive
	s.tr = transport.NewSocketTransport(conn, cb, maxReadCount, keepAlive)
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

func (s *Stream) Authenticated() bool {
	s.RLock()
	defer s.RUnlock()
	return s.authenticated
}

func (s *Stream) Secured() bool {
	s.RLock()
	defer s.RUnlock()
	return s.secured
}

func (s *Stream) Compressed() bool {
	s.RLock()
	defer s.RUnlock()
	return s.compressed
}

func (s *Stream) SendElements(elems []*xml.Element) {
	for _, e := range elems {
		s.SendElement(e)
	}
}

func (s *Stream) SendElement(elem *xml.Element) {
	s.writeCh <- elem
}

func (s *Stream) initializeAuthenticators() {
	for _, a := range s.cfg.SASL {
		switch strings.ToLower(a) {
		case "plain":
			s.authenticators = append(s.authenticators, newPlainAuthenticator(s))
		default:
			break
		}
	}
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
	// validate stream element
	if err := s.validateStreamElement(elem); err != nil {
		s.disconnectWithStreamError(err)
		return
	}
	// assign stream domain
	s.Lock()
	s.domain = elem.To()
	s.Unlock()

	// open stream
	s.openStreamElement()

	features := xml.NewMutableElementName("stream:features")
	features.SetAttribute("xmlns:stream", streamNamespace)
	features.SetAttribute("version", "1.0")

	if !s.Authenticated() {
		// attach TLS feature
		tlsEnabled := s.cfg.TLS.Enabled
		tlsRequired := s.cfg.TLS.Required

		if !s.Secured() && tlsEnabled {
			startTLS := xml.NewMutableElementName("starttls")
			startTLS.SetNamespace("urn:ietf:params:xml:ns:xmpp-tls")
			if tlsRequired {
				startTLS.AppendElement(xml.NewElementName("required"))
			}
			features.AppendElement(startTLS.Copy())
		}

		// attach SASL mechanisms
		shouldOfferSASL := !tlsEnabled || (!tlsRequired || (tlsRequired && s.Secured()))

		if shouldOfferSASL && len(s.authenticators) > 0 {
			mechanisms := xml.NewMutableElementName("mechanisms")
			mechanisms.SetNamespace(saslNamespace)
			for _, athr := range s.authenticators {
				// don't offset authenticators with channel binding on an unsecure stream
				if athr.UsesChannelBinding() && !s.Secured() {
					continue
				}
				mechanism := xml.NewMutableElementName("mechanism")
				mechanism.SetText(athr.Mechanism())
				mechanisms.AppendElement(mechanism.Copy())
			}
			features.AppendElement(mechanisms.Copy())
		}

		// allow In-band registration over encrypted stream only
		allowRegistration := !tlsEnabled || (tlsEnabled && s.Secured())

		if s.cfg.ModRegistration.Enabled && allowRegistration {
			registerFeature := xml.NewElementNamespace("register", "http://jabber.org/features/iq-register")
			features.AppendElement(registerFeature.Copy())
		}
		s.setState(connected)

	} else {
		// attach compression feature
		if !s.Compressed() && s.cfg.Compression.Enabled {
			compression := xml.NewMutableElementNamespace("compression", "http://jabber.org/features/compress")
			method := xml.NewMutableElementName("method")
			method.SetText("zlib")
			compression.AppendElement(method.Copy())
			features.AppendElement(compression.Copy())
		}
		session := xml.NewElementNamespace("session", "urn:ietf:params:xml:ns:xmpp-session")
		features.AppendElement(session)

		bind := xml.NewElementNamespace("bind", "urn:ietf:params:xml:ns:xmpp-bind")
		features.AppendElement(bind)

		s.setState(authenticated)
	}
	s.writeElement(features.Copy())
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
				s.disconnectWithStreamError(ErrInvalidXML)
			}

		case err := <-s.discCh:
			if strmErr, ok := err.(Error); ok {
				s.disconnectWithStreamError(strmErr)
			} else {
				if err != io.EOF {
					log.Errorf("%v", err)
				}
				s.disconnect(false)
			}
		}
	}
}

func (s *Stream) validateStreamElement(elem *xml.Element) Error {
	if elem.Name() != "stream:stream" {
		return ErrUnsupportedStanzaType
	}
	to := elem.To()
	knownHost := false
	if len(to) > 0 {
		for i := 0; i < len(s.cfg.Domains); i++ {
			if s.cfg.Domains[i] == to {
				knownHost = true
				break
			}
		}
	}
	if !knownHost {
		return ErrHostUnknown
	}
	if elem.Namespace() != s.streamDefaultNamespace() || elem.Attribute("xmlns:stream") != streamNamespace {
		return ErrInvalidNamespace
	}
	if elem.Version() != "1.0" {
		return ErrUnsupportedVersion
	}
	return nil
}

func (s *Stream) openStreamElement() {
	ops := xml.NewMutableElementName("stream:stream")
	ops.SetAttribute("xmlns", s.streamDefaultNamespace())
	ops.SetAttribute("xmlns:stream", streamNamespace)
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

func (s *Stream) disconnectWithStreamError(err Error) {
	if s.state() == connecting {
		s.openStreamElement()
	}
	s.writeElementAndWait(err.Element())
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
