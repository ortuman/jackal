/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import (
	"bytes"
	"crypto/tls"
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

const (
	streamNamespace = "http://etherx.jabber.org/streams"
	tlsNamespace    = "urn:ietf:params:xml:ns:xmpp-tls"
)

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

	authrs      []authenticator
	activeAuthr authenticator

	writeCh chan []byte
	readCh  chan []byte
	discCh  chan error
}

func NewStreamSocket(id string, conn net.Conn, config *config.Server) *Stream {
	s := &Stream{
		cfg:     config,
		id:      id,
		parser:  xml.NewParser(),
		st:      connecting,
		writeCh: make(chan []byte, 32),
		readCh:  make(chan []byte, 1),
		discCh:  make(chan error, 1),
	}
	// assign default domain
	s.domain = s.cfg.Domains[0]

	// initialize authenticators
	s.initializeAuthenticators()

	maxReadCount := config.Transport.MaxStanzaSize
	keepAlive := config.Transport.KeepAlive
	s.tr = transport.NewSocketTransport(conn, maxReadCount, keepAlive)

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
	s.writeCh <- []byte(elem.XML(true))
}

func (s *Stream) initializeAuthenticators() {
	for _, a := range s.cfg.SASL {
		switch strings.ToLower(a) {
		case "plain":
			s.authrs = append(s.authrs, newPlainAuthenticator(s))
		default:
			break
		}
	}
}

func (s *Stream) handleElement(elem *xml.Element) {
	switch s.state() {
	case connecting:
		s.handleConnecting(elem)
	case connected:
		s.handleConnected(elem)
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

		if shouldOfferSASL && len(s.authrs) > 0 {
			mechanisms := xml.NewMutableElementName("mechanisms")
			mechanisms.SetNamespace(saslNamespace)
			for _, athr := range s.authrs {
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

func (s *Stream) handleConnected(elem *xml.Element) {
	switch elem.Name() {
	case "starttls":
		if len(elem.Namespace()) > 0 && elem.Namespace() != tlsNamespace {
			s.disconnectWithStreamError(ErrInvalidNamespace)
			return
		}
		s.proceedStartTLS()

	case "auth":
		if elem.Namespace() != saslNamespace {
			s.disconnectWithStreamError(ErrInvalidNamespace)
			return
		}
		s.startAuthentication(elem)
	}
}

func (s *Stream) proceedStartTLS() {
	if s.Secured() {
		s.disconnectWithStreamError(ErrNotAuthorized)
		return
	}
	cer, err := tls.LoadX509KeyPair(s.cfg.TLS.CertFile, s.cfg.TLS.PrivKeyFile)
	if err != nil {
		log.Errorf("%v", err)
		s.writeElementAndWait(xml.NewElementNamespace("failure", tlsNamespace))
		s.disconnect(true)
		return
	}
	s.writeElementAndWait(xml.NewElementNamespace("proceed", tlsNamespace))

	cfg := &tls.Config{
		ServerName:   s.Domain(),
		Certificates: []tls.Certificate{cer},
	}
	s.tr.StartTLS(cfg)
}

func (s *Stream) startAuthentication(elem *xml.Element) {
	mechanism := elem.Attribute("mechanism")
	for _, authr := range s.authrs {
		if authr.Mechanism() == mechanism {
			if err := authr.ProcessElement(elem); err != nil {
				return
			}
			if authr.Authenticated() {
				s.finishAuthentication(authr.Username())
			} else {
				s.activeAuthr = authr
				s.setState(authenticating)
			}
			return
		}
	}

	// ...mechanism not found...
	failure := xml.NewMutableElementNamespace("failure", saslNamespace)
	failure.AppendElement(xml.NewElementName("invalid-mechanism"))
	s.writeElement(failure.Copy())
}

func (s *Stream) continueAuthentication(elem *xml.Element, authr authenticator) {
}

func (s *Stream) finishAuthentication(username string) {
	if s.activeAuthr != nil {
		s.activeAuthr.Reset()
		s.activeAuthr = nil
	}
	s.Lock()
	s.username = username
	s.authenticated = true
	s.Unlock()

	Manager().AuthenticateStream(s)

	s.setState(connecting)
}

func (s *Stream) failAuthentication(elem *xml.Element) {
	failure := xml.NewMutableElementNamespace("failure", saslNamespace)
	failure.AppendElement(elem)
	s.writeElement(failure.Copy())

	if s.activeAuthr != nil {
		s.activeAuthr.Reset()
		s.activeAuthr = nil
	}
	s.setState(connected)
}

func (s *Stream) loop() {
	s.doRead() // start reading transport...
	for {
		// stop looping after disconnecting stream
		if s.state() == disconnected {
			return
		}

		select {
		case b := <-s.writeCh:
			s.writeBytes(b)

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
				s.doRead() // keep reading transport...

			} else { // XML parsing error
				log.Errorf("%v", err)
				s.disconnectWithStreamError(ErrInvalidXML)
			}

		case err := <-s.discCh:
			if strmErr, ok := err.(Error); ok {
				s.disconnectWithStreamError(strmErr)
			} else {
				if err != transport.ErrRemotePeerClosedTransport {
					log.Errorf("%v", err)
				}
				s.disconnect(false)
			}
		}
	}
}

func (s *Stream) doRead() {
	go func() {
		b, err := s.tr.Read()
		switch err {
		case nil:
			log.Debugf("RECV: %s", string(b))
			s.readCh <- b

		case transport.ErrServerClosedTransport:
			return

		default:
			s.discCh <- err
		}
	}()
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
	s.writeBytes([]byte(elem.XML(true)))
}

func (s *Stream) writeElementAndWait(elem *xml.Element) {
	s.writeBytesAndWait([]byte(elem.XML(true)))
}

func (s *Stream) writeBytes(b []byte) {
	log.Debugf("SEND: %s", string(b))
	s.tr.Write(b)
}

func (s *Stream) writeBytesAndWait(b []byte) {
	log.Debugf("SEND: %s", string(b))
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
