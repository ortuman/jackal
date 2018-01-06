/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/config"
	streamerrors "github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream/transport"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
)

type streamState uint8

const (
	connecting streamState = iota
	connected
	authenticating
	authenticated
	sessionStarted
	disconnected
)

const (
	streamNamespace           = "http://etherx.jabber.org/streams"
	tlsNamespace              = "urn:ietf:params:xml:ns:xmpp-tls"
	compressProtocolNamespace = "http://jabber.org/protocol/compress"
	bindNamespace             = "urn:ietf:params:xml:ns:xmpp-bind"
	sessionNamespace          = "urn:ietf:params:xml:ns:xmpp-session"
)

var (
	errResourceNotFound = errors.New("resource not found")
	errNotAuthenticated = errors.New("user not authenticated")
)

type moduleRouter struct {
	strm *Stream
}

type Stream struct {
	sync.RWMutex
	cfg           *config.Server
	connected     uint32
	tr            transport.Transport
	parser        *xml.Parser
	state         streamState
	id            string
	username      string
	domain        string
	resource      string
	jid           *xml.JID
	secured       bool
	authenticated bool
	compressed    bool
	active        bool
	available     bool
	priority      int8

	authrs      []authenticator
	activeAuthr authenticator

	iqHandlers []IQHandler

	roster   *Roster
	register *XEPRegister
	ping     *XEPPing
	offline  *Offline

	writeCh chan []byte
	readCh  chan *xml.Element
	discCh  chan error
}

func NewStreamSocket(id string, conn net.Conn, config *config.Server) *Stream {
	s := &Stream{
		cfg:     config,
		id:      id,
		parser:  xml.NewParser(),
		state:   connecting,
		writeCh: make(chan []byte, 32),
		readCh:  make(chan *xml.Element),
		discCh:  make(chan error),
	}

	// assign default domain
	s.domain = s.cfg.Domains[0]
	s.jid, _ = xml.NewJID("", s.domain, "", true)

	bufferSize := config.Transport.BufferSize
	keepAlive := config.Transport.KeepAlive
	s.tr = transport.NewSocketTransport(conn, bufferSize, keepAlive)

	// initialize authenticators
	s.initializeAuthenticators()

	// initialize XEPs
	s.initializeXEPs()

	if config.Transport.ConnectTimeout > 0 {
		s.startConnectTimeoutTimer(config.Transport.ConnectTimeout)
	}
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

func (s *Stream) JID() *xml.JID {
	s.RLock()
	defer s.RUnlock()
	return s.jid
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

func (s *Stream) Active() bool {
	s.RLock()
	defer s.RUnlock()
	return s.active
}

func (s *Stream) Available() bool {
	s.RLock()
	defer s.RUnlock()
	return s.available
}

func (s *Stream) RequestedRoster() bool {
	if s.roster != nil {
		return s.roster.RequestedRoster()
	}
	return false
}

func (s *Stream) Priority() int8 {
	s.RLock()
	defer s.RUnlock()
	return s.priority
}

func (s *Stream) ChannelBindingBytes(mechanism config.ChannelBindingMechanism) []byte {
	return s.tr.ChannelBindingBytes(mechanism)
}

func (s *Stream) SendElement(serializable xml.Serializable) {
	s.writeCh <- []byte(serializable.XML(true))
}

func (s *Stream) Disconnect(err error) {
	s.discCh <- err
}

func (s *Stream) initializeAuthenticators() {
	for _, a := range s.cfg.SASL {
		switch a {
		case "plain":
			s.authrs = append(s.authrs, newPlainAuthenticator(s))
		case "digest_md5":
			s.authrs = append(s.authrs, newDigestMD5(s))
		case "scram_sha_1":
			s.authrs = append(s.authrs, newScram(s, sha1ScramType, false))
			s.authrs = append(s.authrs, newScram(s, sha1ScramType, true))

		case "scram_sha_256":
			s.authrs = append(s.authrs, newScram(s, sha256ScramType, false))
			s.authrs = append(s.authrs, newScram(s, sha256ScramType, true))
		}
	}
}

func (s *Stream) initializeXEPs() {
	// Roster (https://xmpp.org/rfcs/rfc3921.html#roster)
	s.roster = newRoster(s)
	s.iqHandlers = append(s.iqHandlers, s.roster)

	// XEP-0030: Service Discovery (https://xmpp.org/extensions/xep-0030.html)
	discoInfo := newXEPDiscoInfo(s)
	s.iqHandlers = append(s.iqHandlers, discoInfo)

	// XEP-0049: Private XML Storage (https://xmpp.org/extensions/xep-0049.html)
	if _, ok := s.cfg.Modules["private"]; ok {
		s.iqHandlers = append(s.iqHandlers, newXEPPrivateStorage(s))
	}

	// XEP-0054: vcard-temp (https://xmpp.org/extensions/xep-0054.html)
	if _, ok := s.cfg.Modules["vcard"]; ok {
		s.iqHandlers = append(s.iqHandlers, newXEPVCard(s))
	}

	// XEP-0077: In-band registration (https://xmpp.org/extensions/xep-0077.html)
	if _, ok := s.cfg.Modules["registration"]; ok {
		s.register = newXEPRegister(&s.cfg.ModRegistration, s)
		s.iqHandlers = append(s.iqHandlers, s.register)
	}

	// XEP-0092: Software Version (https://xmpp.org/extensions/xep-0092.html)
	if _, ok := s.cfg.Modules["version"]; ok {
		s.iqHandlers = append(s.iqHandlers, newXEPVersion(&s.cfg.ModVersion, s))
	}

	// XEP-0199: XMPP Ping (https://xmpp.org/extensions/xep-0199.html)
	if _, ok := s.cfg.Modules["ping"]; ok {
		s.ping = newXEPPing(&s.cfg.ModPing, s)
		s.iqHandlers = append(s.iqHandlers, s.ping)
	}

	// register server disco info identities
	identities := []DiscoIdentity{{
		Category: "server",
		Type:     "im",
		Name:     s.cfg.ID,
	}}
	discoInfo.SetIdentities(identities)

	// register disco info features
	var features []string
	for _, iqHandler := range s.iqHandlers {
		features = append(features, iqHandler.AssociatedNamespaces()...)
	}

	// XEP-0160: Offline message storage (https://xmpp.org/extensions/xep-0160.html)
	if _, ok := s.cfg.Modules["offline"]; ok {
		s.offline = newOffline(&s.cfg.ModOffline, s)
		features = append(features, s.offline.AssociatedNamespaces()...)
	}
	discoInfo.SetFeatures(features)
}

func (s *Stream) startConnectTimeoutTimer(timeoutInSeconds int) {
	go func() {
		tr := time.NewTimer(time.Second * time.Duration(timeoutInSeconds))
		<-tr.C
		if atomic.LoadUint32(&s.connected) == 0 {
			// connection timeout...
			s.discCh <- streamerrors.ErrConnectionTimeout
		}
	}()
}

func (s *Stream) handleElement(elem *xml.Element) {
	switch s.state {
	case connecting:
		s.handleConnecting(elem)
	case connected:
		s.handleConnected(elem)
	case authenticated:
		s.handleAuthenticated(elem)
	case authenticating:
		s.handleAuthenticating(elem)
	case sessionStarted:
		s.handleSessionStarted(elem)
	default:
		break
	}
}

func (s *Stream) handleConnecting(elem *xml.Element) {
	// activate 'connected' flag
	atomic.StoreUint32(&s.connected, 1)

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
		tlsEnabled := s.cfg.TLS != nil
		tlsRequired := s.cfg.TLS != nil && s.cfg.TLS.Required

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

		if _, ok := s.cfg.Modules["offline"]; ok && allowRegistration {
			registerFeature := xml.NewElementNamespace("register", "http://jabber.org/features/iq-register")
			features.AppendElement(registerFeature.Copy())
		}
		s.state = connected

	} else {
		// attach compression feature
		if !s.Compressed() && s.cfg.Compression != nil {
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

		s.state = authenticated
	}
	s.writeElement(features.Copy())
}

func (s *Stream) handleConnected(elem *xml.Element) {
	switch elem.Name() {
	case "starttls":
		if len(elem.Namespace()) > 0 && elem.Namespace() != tlsNamespace {
			s.disconnectWithStreamError(streamerrors.ErrInvalidNamespace)
			return
		}
		s.proceedStartTLS()

	case "auth":
		if elem.Namespace() != saslNamespace {
			s.disconnectWithStreamError(streamerrors.ErrInvalidNamespace)
			return
		}
		s.startAuthentication(elem)

	case "iq":
		stanza, _, err := s.buildStanza(elem)
		if err != nil {
			s.handleElementError(elem, err)
			return
		}
		iq := stanza.(*xml.IQ)

		if s.register != nil && s.register.MatchesIQ(iq) {
			s.register.ProcessIQ(iq)
			return

		} else if iq.FindElementNamespace("query", "jabber:iq:auth") != nil {
			// don't allow non-SASL authentication
			s.writeElement(iq.ServiceUnavailableError())
			return
		}
		fallthrough

	case "message", "presence":
		s.disconnectWithStreamError(streamerrors.ErrNotAuthorized)

	default:
		s.disconnectWithStreamError(streamerrors.ErrUnsupportedStanzaType)
	}
}

func (s *Stream) handleAuthenticating(elem *xml.Element) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(streamerrors.ErrInvalidNamespace)
		return
	}
	authr := s.activeAuthr
	s.continueAuthentication(elem, authr)
	if authr.Authenticated() {
		s.finishAuthentication(authr.Username())
	}
}

func (s *Stream) handleAuthenticated(elem *xml.Element) {
	switch elem.Name() {
	case "compress":
		if elem.Namespace() != compressProtocolNamespace {
			s.disconnectWithStreamError(streamerrors.ErrUnsupportedStanzaType)
			return
		}
		s.compress(elem)

	case "iq":
		stanza, _, err := s.buildStanza(elem)
		if err != nil {
			s.handleElementError(elem, err)
			return
		}
		iq := stanza.(*xml.IQ)

		if len(s.Resource()) == 0 { // expecting bind
			s.bindResource(iq)
		} else { // expecting session
			s.startSession(iq)
		}

	default:
		s.disconnectWithStreamError(streamerrors.ErrUnsupportedStanzaType)
	}
}

func (s *Stream) handleSessionStarted(elem *xml.Element) {
	// reset ping timer deadline
	if s.ping != nil {
		s.ping.ResetDeadline()
	}

	stanza, toJID, err := s.buildStanza(elem)
	if err != nil {
		s.handleElementError(elem, err)
		return
	}
	if s.isValidDomain(toJID.Domain()) {
		// local stanza
		s.processStanza(stanza)
	} else if s.isComponentDomain(toJID.Domain()) {
		// component (MUC, pubsub, etc.)
		s.processComponentStanza(stanza)
	} else {
		// S2S
	}
}

func (s *Stream) proceedStartTLS() {
	if s.Secured() {
		s.disconnectWithStreamError(streamerrors.ErrNotAuthorized)
		return
	}
	cer, err := tls.LoadX509KeyPair(s.cfg.TLS.CertFile, s.cfg.TLS.PrivKeyFile)
	if err != nil {
		log.Error(err)
		s.writeElement(xml.NewElementNamespace("failure", tlsNamespace))
		s.disconnect(true)
		return
	}
	s.writeElement(xml.NewElementNamespace("proceed", tlsNamespace))

	cfg := &tls.Config{
		ServerName:   s.Domain(),
		Certificates: []tls.Certificate{cer},
	}
	s.tr.StartTLS(cfg)

	s.Lock()
	s.secured = true
	s.Unlock()

	log.Infof("secured stream... id: %s", s.id)

	s.restart()
}

func (s *Stream) compress(elem *xml.Element) {
	if s.Compressed() {
		s.disconnectWithStreamError(streamerrors.ErrUnsupportedStanzaType)
		return
	}
	method := elem.FindElement("method")
	if method == nil || method.TextLen() == 0 {
		failure := xml.NewMutableElementNamespace("failure", compressProtocolNamespace)
		failure.AppendElement(xml.NewElementName("setup-failed"))
		s.writeElement(failure.Copy())
		return
	}
	if method.Text() != "zlib" {
		failure := xml.NewMutableElementNamespace("failure", compressProtocolNamespace)
		failure.AppendElement(xml.NewElementName("unsupported-method"))
		s.writeElement(failure.Copy())
		return
	}
	compressed := xml.NewElementNamespace("compressed", compressProtocolNamespace)
	s.writeElement(compressed)

	s.tr.EnableCompression(s.cfg.Compression.Level)
	s.Lock()
	s.compressed = true
	s.Unlock()

	log.Infof("compressed stream... id: %s", s.id)

	s.restart()
}

func (s *Stream) startAuthentication(elem *xml.Element) {
	mechanism := elem.Attribute("mechanism")
	for _, authr := range s.authrs {
		if authr.Mechanism() == mechanism {
			if err := s.continueAuthentication(elem, authr); err != nil {
				return
			}
			if authr.Authenticated() {
				s.finishAuthentication(authr.Username())
			} else {
				s.activeAuthr = authr
				s.state = authenticating
			}
			return
		}
	}

	// ...mechanism not found...
	failure := xml.NewMutableElementNamespace("failure", saslNamespace)
	failure.AppendElement(xml.NewElementName("invalid-mechanism"))
	s.writeElement(failure.Copy())
}

func (s *Stream) continueAuthentication(elem *xml.Element, authr authenticator) error {
	err := authr.ProcessElement(elem)
	if saslErr, ok := err.(saslError); ok {
		s.failAuthentication(saslErr.Element())
	} else if err != nil {
		log.Error(err)
		s.failAuthentication(errSASLTemporaryAuthFailure.(saslError).Element())
	}
	return err
}

func (s *Stream) finishAuthentication(username string) {
	if s.activeAuthr != nil {
		s.activeAuthr.Reset()
		s.activeAuthr = nil
	}
	s.Lock()
	s.username = username
	s.authenticated = true
	s.jid, _ = xml.NewJID(s.username, s.domain, "", true)
	s.Unlock()

	Manager().AuthenticateStream(s)
	s.restart()
}

func (s *Stream) failAuthentication(elem *xml.Element) {
	failure := xml.NewMutableElementNamespace("failure", saslNamespace)
	failure.AppendElement(elem)
	s.writeElement(failure.Copy())

	if s.activeAuthr != nil {
		s.activeAuthr.Reset()
		s.activeAuthr = nil
	}
	s.state = connected
}

func (s *Stream) bindResource(iq *xml.IQ) {
	bind := iq.FindElementNamespace("bind", bindNamespace)
	if bind == nil {
		s.writeElement(iq.NotAllowedError())
		return
	}
	var resource string
	if resourceElem := bind.FindElement("resource"); resourceElem != nil {
		resource = resourceElem.Text()
	} else {
		resource = uuid.New()
	}
	// try binding...
	if !s.isResourceAvailable(resource) {
		s.writeElement(iq.ConflictError())
		return
	}
	userJID, err := xml.NewJID(s.Username(), s.Domain(), resource, false)
	if err != nil {
		s.writeElement(iq.BadRequestError())
		return
	}
	s.Lock()
	s.resource = resource
	s.jid = userJID
	s.Unlock()

	log.Infof("binded resource %s... (%s)", s.Resource(), s.Username())

	//...notify successful binding
	result := xml.NewMutableIQType(iq.ID(), xml.ResultType)

	binded := xml.NewMutableElementNamespace("bind", bindNamespace)
	jid := xml.NewMutableElementName("jid")
	jid.SetText(s.Username() + "@" + s.Domain() + "/" + s.Resource())
	binded.AppendElement(jid.Copy())
	result.AppendElement(binded.Copy())

	s.writeElement(result.Copy())
}

func (s *Stream) startSession(iq *xml.IQ) {
	sess := iq.FindElementNamespace("session", sessionNamespace)
	if sess == nil {
		s.writeElement(iq.NotAllowedError())
		return
	}
	s.writeElement(iq.ResultIQ())

	if s.ping != nil {
		s.ping.StartPinging()
	}
	s.state = sessionStarted
	s.active = true
}

func (s *Stream) processStanza(stanza xml.Serializable) {
	if iq, ok := stanza.(*xml.IQ); ok {
		s.processIQ(iq)
	} else if presence, ok := stanza.(*xml.Presence); ok {
		s.processPresence(presence)
	} else if message, ok := stanza.(*xml.Message); ok {
		s.processMessage(message)
	}
}

func (s *Stream) processComponentStanza(stanza xml.Serializable) {
}

func (s *Stream) processIQ(iq *xml.IQ) {
	toJid := iq.ToJID()
	if toJid.IsFull() {
		if err := s.sendElement(iq, toJid); err == errResourceNotFound {
			resp := iq.MutableCopy()
			resp.SetFrom(toJid.String())
			resp.SetTo(s.JID().String())
			s.SendElement(resp.ServiceUnavailableError())
		}
		return
	}

	for _, handler := range s.iqHandlers {
		if !handler.MatchesIQ(iq) {
			continue
		}
		handler.ProcessIQ(iq)
		return
	}

	// ...IQ not handled...
	if iq.IsGet() || iq.IsSet() {
		s.writeElement(iq.ServiceUnavailableError())
	}
}

func (s *Stream) processPresence(presence *xml.Presence) {
	toJid := presence.ToJID()
	if toJid.IsFull() {
		s.sendElement(presence, toJid)
		return
	}
	if toJid.IsBare() && toJid.Node() != s.Username() {
		if s.roster != nil {
			s.roster.ProcessPresence(presence)
		}
		return
	}
	// set resource priority & availability
	s.Lock()
	defer s.Unlock()

	s.priority = presence.Priority()
	s.available = true

	// deliver pending approval notifications
	if s.roster != nil {
		s.roster.DeliverPendingApprovalNotifications()
	}

	// deliver offline messages
	if s.offline != nil && s.priority >= 0 {
		s.offline.DeliverOfflineMessages()
	}
}

func (s *Stream) processMessage(message *xml.Message) {
	err := s.sendElement(message, message.ToJID())
	switch err {
	case errNotAuthenticated:
		if s.offline != nil {
			s.offline.ArchiveMessage(message)
		}
	case errResourceNotFound:
		resp := message.MutableCopy()
		resp.SetFrom(message.ToJID().String())
		resp.SetTo(s.JID().String())
		s.SendElement(resp.ServiceUnavailableError())
	}
}

func (s *Stream) restart() {
	s.state = connecting
	s.parser = xml.NewParser()
}

func (s *Stream) loop() {
	s.doRead() // start reading transport...
	for {
		// stop looping after disconnecting stream
		if s.state == disconnected {
			return
		}

		select {
		case b := <-s.writeCh:
			s.writeBytes(b)

		case e := <-s.readCh:
			s.handleElement(e)
			if s.state != disconnected {
				s.doRead() // keep reading transport...
			}

		case err := <-s.discCh:
			switch err {
			case nil:
				s.disconnect(false)
			default:
				if strmErr, ok := err.(*streamerrors.StreamError); ok {
					s.disconnectWithStreamError(strmErr)
				} else {
					log.Error(err)
					s.disconnect(false)
				}
			}
		}
	}
}

func (s *Stream) doRead() {
	go func() {
		if e, err := s.parser.ParseElement(s.tr); e != nil && err == nil {
			if log.Level() >= config.DebugLevel {
				log.Debugf("RECV: %s", e.XML(true))
			}
			s.readCh <- e
		} else if err != nil {
			switch err {
			case nil:
				break
			case io.EOF, io.ErrUnexpectedEOF, xml.ErrStreamClosedByPeer:
				s.discCh <- nil
			default:
				log.Error(err)
				s.discCh <- streamerrors.ErrInvalidXML
			}
		}
	}()
}

func (s *Stream) openStreamElement() {
	ops := xml.NewMutableElementName("stream:stream")
	ops.SetAttribute("xmlns", s.streamDefaultNamespace())
	ops.SetAttribute("xmlns:stream", streamNamespace)
	ops.SetAttribute("id", uuid.New())
	ops.SetAttribute("from", s.Domain())
	ops.SetAttribute("version", "1.0")

	s.writeBytes([]byte(`<?xml version="1.0"?>`))
	s.writeBytes([]byte(ops.XML(false)))
}

func (s *Stream) buildStanza(elem *xml.Element) (xml.Serializable, *xml.JID, error) {
	if err := s.validateNamespace(elem); err != nil {
		return nil, nil, err
	}
	fromJID, toJID, err := s.validateAddresses(elem)
	if err != nil {
		return nil, nil, err
	}
	switch elem.Name() {
	case "iq":
		iq, err := xml.NewIQ(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, nil, xml.ErrBadRequest
		}
		return iq, iq.ToJID(), nil

	case "presence":
		presence, err := xml.NewPresence(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, nil, xml.ErrBadRequest
		}
		return presence, presence.ToJID(), nil

	case "message":
		message, err := xml.NewMessage(elem, fromJID, toJID)
		if err != nil {
			log.Error(err)
			return nil, nil, xml.ErrBadRequest
		}
		return message, message.ToJID(), nil
	}
	return nil, nil, streamerrors.ErrUnsupportedStanzaType
}

func (s *Stream) handleElementError(elem *xml.Element, err error) {
	if streamErr, ok := err.(*streamerrors.StreamError); ok {
		s.disconnectWithStreamError(streamErr)
	} else if stanzaErr, ok := err.(*xml.StanzaError); ok {
		s.writeElement(elem.ToError(stanzaErr))
	} else {
		log.Error(err)
	}
}

func (s *Stream) validateStreamElement(elem *xml.Element) *streamerrors.StreamError {
	if elem.Name() != "stream:stream" {
		return streamerrors.ErrUnsupportedStanzaType
	}
	to := elem.To()
	if len(to) > 0 && !s.isValidDomain(to) {
		return streamerrors.ErrHostUnknown
	}
	if elem.Namespace() != s.streamDefaultNamespace() || elem.Attribute("xmlns:stream") != streamNamespace {
		return streamerrors.ErrInvalidNamespace
	}
	if elem.Version() != "1.0" {
		return streamerrors.ErrUnsupportedVersion
	}
	return nil
}

func (s *Stream) validateNamespace(elem *xml.Element) *streamerrors.StreamError {
	ns := elem.Namespace()
	if len(ns) == 0 || ns == s.streamDefaultNamespace() {
		return nil
	}
	return streamerrors.ErrInvalidNamespace
}

func (s *Stream) validateAddresses(elem *xml.Element) (fromJID *xml.JID, toJID *xml.JID, err error) {
	// validate from JID
	from := elem.From()
	if len(from) > 0 && !s.isValidFrom(from) {
		return nil, nil, streamerrors.ErrInvalidFrom
	}
	fromJID = s.JID()

	// validate to JID
	to := elem.To()
	if len(to) > 0 {
		toJID, err = xml.NewJIDString(elem.To(), false)
		if err != nil {
			return nil, nil, xml.ErrJidMalformed
		}
	} else {
		toJID, err = xml.NewJID("", s.Domain(), "", true)
	}
	return
}

func (s *Stream) isValidFrom(from string) bool {
	validFrom := false
	j, err := xml.NewJIDString(from, false)
	if err == nil && j != nil {
		node := j.Node()
		domain := j.Domain()
		resource := j.Resource()

		userJID := s.JID()
		validFrom = node == userJID.Node() && domain == userJID.Domain()
		if len(resource) > 0 {
			validFrom = validFrom && resource == userJID.Resource()
		}
	}
	return validFrom
}

func (s *Stream) isValidDomain(domain string) bool {
	for i := 0; i < len(s.cfg.Domains); i++ {
		if s.cfg.Domains[i] == domain {
			return true
		}
	}
	return false
}

func (s *Stream) isComponentDomain(domain string) bool {
	return false
}

func (s *Stream) streamDefaultNamespace() string {
	switch s.cfg.Type {
	case config.C2S:
		return "jabber:client"
	case config.S2S:
		return "jabber:server"
	}
	// should not be reached
	return ""
}

func (s *Stream) writeElement(elem xml.Serializable) {
	s.writeBytes([]byte(elem.XML(true)))
}

func (s *Stream) writeBytes(b []byte) {
	if log.Level() >= config.DebugLevel {
		log.Debugf("SEND: %s", string(b))
	}
	s.tr.Write(b)
}

func (s *Stream) disconnectWithStreamError(err *streamerrors.StreamError) {
	if s.state == connecting {
		s.openStreamElement()
	}
	s.writeElement(err.Element())
	s.disconnect(true)
}

func (s *Stream) disconnect(closeStream bool) {
	if closeStream {
		s.tr.Write([]byte("</stream:stream>"))
	}
	s.tr.Close()

	s.state = disconnected

	Manager().UnregisterStream(s)
}

func (s *Stream) isResourceAvailable(resource string) bool {
	strms := Manager().UserStreams(s.Username())
	for _, strm := range strms {
		if strm.Resource() == resource {
			return false
		}
	}
	return true
}

func (s *Stream) sendElement(serializable xml.Serializable, to *xml.JID) error {
	recipients := Manager().UserStreams(to.Node())
	if len(recipients) == 0 {
		return errNotAuthenticated
	}
	if to.IsFull() {
		for _, strm := range recipients {
			if strm.Resource() == to.Resource() {
				strm.SendElement(serializable)
				return nil
			}
		}
		return errResourceNotFound

	} else {
		switch serializable.(type) {
		case *xml.Message:
			// send to highest priority stream
			strm := recipients[0]
			highestPriority := strm.Priority()
			for i := 1; i < len(recipients); i++ {
				if recipients[i].Priority() > highestPriority {
					strm = recipients[i]
				}
			}
			strm.SendElement(serializable)

		default:
			// broadcast to all streams
			for _, strm := range recipients {
				strm.SendElement(serializable)
			}
		}
	}
	return nil
}
