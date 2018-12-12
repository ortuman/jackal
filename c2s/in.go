/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"crypto/tls"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/auth"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/session"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/transport/compress"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
)

const (
	connecting uint32 = iota
	connected
	authenticating
	authenticated
	sessionStarted
	disconnected
)

type inStream struct {
	cfg            *streamConfig
	router         *router.Router
	mods           *module.Modules
	comps          *component.Components
	sess           *session.Session
	id             string
	connectTm      *time.Timer
	state          uint32
	ctx            *stream.Context
	authenticators []auth.Authenticator
	activeAuth     auth.Authenticator
	actorCh        chan func()
	iqResultCh     chan xmpp.Stanza

	mu            sync.RWMutex
	jid           *jid.JID
	secured       bool
	compressed    bool
	authenticated bool
	presence      *xmpp.Presence
}

func newStream(id string, config *streamConfig, mods *module.Modules, comps *component.Components, router *router.Router) stream.C2S {
	s := &inStream{
		cfg:        config,
		router:     router,
		mods:       mods,
		comps:      comps,
		id:         id,
		ctx:        stream.NewContext(),
		actorCh:    make(chan func(), streamMailboxSize),
		iqResultCh: make(chan xmpp.Stanza, iqResultMailboxSize),
	}

	// Initialize stream context
	secured := !(config.transport.Type() == transport.Socket)
	s.setSecured(secured)
	s.setJID(&jid.JID{})

	// Initialize authenticators
	s.initializeAuthenticators()

	// Start c2s session
	s.restartSession()

	if config.connectTimeout > 0 {
		s.connectTm = time.AfterFunc(config.connectTimeout, s.connectTimeout)
	}
	go s.loop()
	go s.doRead() // Start reading...

	return s
}

// ID returns stream identifier.
func (s *inStream) ID() string {
	return s.id
}

// context returns stream associated context.
func (s *inStream) Context() *stream.Context {
	return s.ctx
}

// Username returns current stream username.
func (s *inStream) Username() string {
	return s.JID().Node()
}

// Domain returns current stream domain.
func (s *inStream) Domain() string {
	return s.JID().Domain()
}

// Resource returns current stream resource.
func (s *inStream) Resource() string {
	return s.JID().Resource()
}

// JID returns current user JID.
func (s *inStream) JID() *jid.JID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jid
}

// IsAuthenticated returns whether or not the XMPP stream has successfully authenticated.
func (s *inStream) IsAuthenticated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authenticated
}

// IsSecured returns whether or not the XMPP stream has been secured using SSL/TLS.
func (s *inStream) IsSecured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.secured
}

// Presence returns last sent presence element.
func (s *inStream) Presence() *xmpp.Presence {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.presence
}

// SendElement sends the given XML element.
func (s *inStream) SendElement(elem xmpp.XElement) {
	if s.getState() == disconnected {
		return
	}
	s.actorCh <- func() { s.writeElement(elem) }
}

// Disconnect disconnects remote peer by closing the underlying TCP socket connection.
func (s *inStream) Disconnect(err error) {
	if s.getState() == disconnected {
		return
	}
	waitCh := make(chan struct{})
	s.actorCh <- func() {
		s.disconnect(err)
		close(waitCh)
	}
	<-waitCh
}

func (s *inStream) initializeAuthenticators() {
	tr := s.cfg.transport
	var authenticators []auth.Authenticator
	for _, a := range s.cfg.sasl {
		switch a {
		case "plain":
			authenticators = append(authenticators, auth.NewPlain(s))

		case "digest_md5":
			authenticators = append(authenticators, auth.NewDigestMD5(s))

		case "scram_sha_1":
			authenticators = append(authenticators, auth.NewScram(s, tr, auth.ScramSHA1, false))
			authenticators = append(authenticators, auth.NewScram(s, tr, auth.ScramSHA1, true))

		case "scram_sha_256":
			authenticators = append(authenticators, auth.NewScram(s, tr, auth.ScramSHA256, false))
			authenticators = append(authenticators, auth.NewScram(s, tr, auth.ScramSHA256, true))
		}
	}
	s.authenticators = authenticators
}

func (s *inStream) connectTimeout() {
	s.actorCh <- func() { s.disconnect(streamerror.ErrConnectionTimeout) }
}

func (s *inStream) handleElement(elem xmpp.XElement) {
	switch s.getState() {
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
	}
}

func (s *inStream) handleConnecting(elem xmpp.XElement) {
	// Cancel connection timeout timer
	if s.connectTm != nil {
		s.connectTm.Stop()
		s.connectTm = nil
	}
	// Assign stream domain if not set yet
	if len(s.Domain()) == 0 {
		j, _ := jid.New("", elem.To(), "", true)
		s.setJID(j)
	}

	// Open stream session
	s.sess.SetJID(s.JID())
	s.sess.Open()

	features := xmpp.NewElementName("stream:features")
	features.SetAttribute("xmlns:stream", streamNamespace)
	features.SetAttribute("version", "1.0")

	if !s.IsAuthenticated() {
		features.AppendElements(s.unauthenticatedFeatures())
		s.setState(connected)
	} else {
		features.AppendElements(s.authenticatedFeatures())
		s.setState(authenticated)
	}
	s.writeElement(features)
}

func (s *inStream) unauthenticatedFeatures() []xmpp.XElement {
	var features []xmpp.XElement

	isSocketTr := s.cfg.transport.Type() == transport.Socket

	if isSocketTr && !s.IsSecured() {
		startTLS := xmpp.NewElementName("starttls")
		startTLS.SetNamespace("urn:ietf:params:xml:ns:xmpp-tls")
		startTLS.AppendElement(xmpp.NewElementName("required"))
		features = append(features, startTLS)
	}

	// Attach SASL mechanisms
	shouldOfferSASL := !isSocketTr || (isSocketTr && s.IsSecured())

	if shouldOfferSASL && len(s.authenticators) > 0 {
		mechanisms := xmpp.NewElementName("mechanisms")
		mechanisms.SetNamespace(saslNamespace)
		for _, athr := range s.authenticators {
			mechanism := xmpp.NewElementName("mechanism")
			mechanism.SetText(athr.Mechanism())
			mechanisms.AppendElement(mechanism)
		}
		features = append(features, mechanisms)
	}

	// Allow In-band registration over encrypted stream only
	allowRegistration := s.IsSecured()

	if reg := s.mods.Register; reg != nil && allowRegistration {
		registerFeature := xmpp.NewElementNamespace("register", "http://jabber.org/features/iq-register")
		features = append(features, registerFeature)
	}
	return features
}

func (s *inStream) authenticatedFeatures() []xmpp.XElement {
	var features []xmpp.XElement

	isSocketTr := s.cfg.transport.Type() == transport.Socket

	// Attach compression feature
	compressionAvailable := isSocketTr && s.cfg.compression.Level != compress.NoCompression

	if !s.isCompressed() && compressionAvailable {
		compression := xmpp.NewElementNamespace("compression", "http://jabber.org/features/compress")
		method := xmpp.NewElementName("method")
		method.SetText("zlib")
		compression.AppendElement(method)
		features = append(features, compression)
	}
	bind := xmpp.NewElementNamespace("bind", "urn:ietf:params:xml:ns:xmpp-bind")
	bind.AppendElement(xmpp.NewElementName("required"))
	features = append(features, bind)

	sessElem := xmpp.NewElementNamespace("session", "urn:ietf:params:xml:ns:xmpp-session")
	features = append(features, sessElem)

	if s.mods.Roster != nil {
		ver := xmpp.NewElementNamespace("ver", "urn:xmpp:features:rosterver")
		features = append(features, ver)
	}
	return features
}

func (s *inStream) handleConnected(elem xmpp.XElement) {
	switch elem.Name() {
	case "starttls":
		s.proceedStartTLS(elem)

	case "auth":
		s.startAuthentication(elem)

	case "iq":
		iq := elem.(*xmpp.IQ)
		if reg := s.mods.Register; reg != nil && reg.MatchesIQ(iq) {
			if s.IsSecured() {
				reg.ProcessIQ(iq, s)
			} else {
				// Channel isn't safe enough to enable a password change
				s.writeElement(iq.NotAuthorizedError())
			}
			return

		} else if iq.Elements().ChildNamespace("query", "jabber:iq:auth") != nil {
			// Don't allow non-SASL authentication
			s.writeElement(iq.ServiceUnavailableError())
			return
		}
		fallthrough

	case "message", "presence":
		s.disconnectWithStreamError(streamerror.ErrNotAuthorized)

	default:
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *inStream) handleAuthenticating(elem xmpp.XElement) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
		return
	}
	authr := s.activeAuth
	s.continueAuthentication(elem, authr)
	if authr.Authenticated() {
		s.finishAuthentication(authr.Username())
	}
}

func (s *inStream) handleAuthenticated(elem xmpp.XElement) {
	switch elem.Name() {
	case "compress":
		if elem.Namespace() != compressProtocolNamespace {
			s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
			return
		}
		s.compress(elem)

	case "iq":
		iq := elem.(*xmpp.IQ)
		if len(s.JID().Resource()) == 0 { // Expecting bind
			s.bindResource(iq)
		} else { // Expecting session
			s.startSession(iq)
		}

	default:
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *inStream) handleSessionStarted(elem xmpp.XElement) {
	// Reset ping timer deadline
	if p := s.mods.Ping; p != nil {
		p.SchedulePing(s)
	}
	stanza, ok := elem.(xmpp.Stanza)
	if !ok {
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
		return
	}
	if comp := s.comps.Get(stanza.ToJID().Domain()); comp != nil { // component stanza?
		switch stanza := stanza.(type) {
		case *xmpp.IQ:
			if di := s.mods.DiscoInfo; di != nil && di.MatchesIQ(stanza) {
				di.ProcessIQ(stanza, s)
				return
			}
			break
		}
		comp.ProcessStanza(stanza, s)
	} else {
		s.processStanza(stanza)
	}
}

func (s *inStream) proceedStartTLS(elem xmpp.XElement) {
	if s.IsSecured() {
		s.disconnectWithStreamError(streamerror.ErrNotAuthorized)
		return
	}
	if len(elem.Namespace()) > 0 && elem.Namespace() != tlsNamespace {
		s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
		return
	}
	s.writeElement(xmpp.NewElementNamespace("proceed", tlsNamespace))

	s.cfg.transport.StartTLS(&tls.Config{Certificates: s.router.Certificates()}, false)
	s.setSecured(true)

	log.Infof("secured stream... id: %s", s.id)
	s.restartSession()
}

func (s *inStream) compress(elem xmpp.XElement) {
	if s.isCompressed() {
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
		return
	}
	method := elem.Elements().Child("method")
	if method == nil || len(method.Text()) == 0 {
		failure := xmpp.NewElementNamespace("failure", compressProtocolNamespace)
		failure.AppendElement(xmpp.NewElementName("setup-failed"))
		s.writeElement(failure)
		return
	}
	if method.Text() != "zlib" {
		failure := xmpp.NewElementNamespace("failure", compressProtocolNamespace)
		failure.AppendElement(xmpp.NewElementName("unsupported-method"))
		s.writeElement(failure)
		return
	}
	s.writeElement(xmpp.NewElementNamespace("compressed", compressProtocolNamespace))

	s.cfg.transport.EnableCompression(s.cfg.compression.Level)
	s.setCompressed(true)

	log.Infof("compressed stream... id: %s", s.id)

	s.restartSession()
}

func (s *inStream) startAuthentication(elem xmpp.XElement) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
		return
	}
	mechanism := elem.Attributes().Get("mechanism")
	for _, authr := range s.authenticators {
		if authr.Mechanism() == mechanism {
			if err := s.continueAuthentication(elem, authr); err != nil {
				return
			}
			if authr.Authenticated() {
				s.finishAuthentication(authr.Username())
			} else {
				s.activeAuth = authr
				s.setState(authenticating)
			}
			return
		}
	}
	// ...mechanism not found...
	failure := xmpp.NewElementNamespace("failure", saslNamespace)
	failure.AppendElement(xmpp.NewElementName("invalid-mechanism"))
	s.writeElement(failure)
}

func (s *inStream) continueAuthentication(elem xmpp.XElement, authr auth.Authenticator) error {
	err := authr.ProcessElement(elem)
	if saslErr, ok := err.(*auth.SASLError); ok {
		s.failAuthentication(saslErr.Element())
	} else if err != nil {
		log.Error(err)
		s.failAuthentication(auth.ErrSASLTemporaryAuthFailure.(*auth.SASLError).Element())
	}
	return err
}

func (s *inStream) finishAuthentication(username string) {
	if s.activeAuth != nil {
		s.activeAuth.Reset()
		s.activeAuth = nil
	}
	j, _ := jid.New(username, s.Domain(), "", true)
	s.setJID(j)
	s.setAuthenticated(true)

	s.restartSession()
}

func (s *inStream) failAuthentication(elem xmpp.XElement) {
	failure := xmpp.NewElementNamespace("failure", saslNamespace)
	failure.AppendElement(elem)
	s.writeElement(failure)

	if s.activeAuth != nil {
		s.activeAuth.Reset()
		s.activeAuth = nil
	}
	s.setState(connected)
}

func (s *inStream) bindResource(iq *xmpp.IQ) {
	bind := iq.Elements().ChildNamespace("bind", bindNamespace)
	if bind == nil {
		s.writeElement(iq.NotAllowedError())
		return
	}
	var resource string
	if resourceElem := bind.Elements().Child("resource"); resourceElem != nil {
		resource = resourceElem.Text()
	} else {
		resource = uuid.New()
	}
	// Try binding...
	var stm stream.C2S
	stms := s.router.UserStreams(s.JID().Node())
	for _, s := range stms {
		if s.Resource() == resource {
			stm = s
		}
	}
	if stm != nil {
		switch s.cfg.resourceConflict {
		case Override:
			// Override the resource with a server-generated resourcepart...
			resource = uuid.New()
		case Replace:
			// Terminate the session of the currently connected client...
			stm.Disconnect(streamerror.ErrResourceConstraint)
		default:
			// Disallow resource binding attempt...
			s.writeElement(iq.ConflictError())
			return
		}
	}
	userJID, err := jid.New(s.Username(), s.Domain(), resource, false)
	if err != nil {
		s.writeElement(iq.BadRequestError())
		return
	}
	s.setJID(userJID)

	s.sess.SetJID(userJID)

	s.router.Bind(s)

	//...notify successful binding
	result := xmpp.NewIQType(iq.ID(), xmpp.ResultType)
	result.SetNamespace(iq.Namespace())

	binded := xmpp.NewElementNamespace("bind", bindNamespace)
	j := xmpp.NewElementName("jid")
	j.SetText(s.Username() + "@" + s.Domain() + "/" + s.Resource())
	binded.AppendElement(j)
	result.AppendElement(binded)

	s.writeElement(result)
}

func (s *inStream) startSession(iq *xmpp.IQ) {
	if len(s.Resource()) == 0 {
		// Not binded yet...
		s.Disconnect(streamerror.ErrNotAuthorized)
		return
	}
	sess := iq.Elements().ChildNamespace("session", sessionNamespace)
	if sess == nil {
		s.writeElement(iq.NotAllowedError())
		return
	}
	s.writeElement(iq.ResultIQ())

	// Start pinging...
	if p := s.mods.Ping; p != nil {
		p.SchedulePing(s)
	}
	s.setState(sessionStarted)
}

func (s *inStream) processStanza(elem xmpp.Stanza) {
	toJID := elem.ToJID()
	if s.isBlockedJID(toJID) { // Blocked JID?
		blocked := xmpp.NewElementNamespace("blocked", blockedErrorNamespace)
		resp := xmpp.NewErrorStanzaFromStanza(elem, xmpp.ErrNotAcceptable, []xmpp.XElement{blocked})
		s.writeElement(resp)
		return
	}
	switch stanza := elem.(type) {
	case *xmpp.Presence:
		s.processPresence(stanza)
	case *xmpp.IQ:
		s.processIQ(stanza)
	case *xmpp.Message:
		s.processMessage(stanza)
	}
}

func (s *inStream) processIQ(iq *xmpp.IQ) {
	toJID := iq.ToJID()

	replyOnBehalf := !toJID.IsFullWithUser() && s.router.IsLocalHost(toJID.Domain())
	if !replyOnBehalf {
		switch s.router.Route(iq) {
		case router.ErrResourceNotFound:
			s.writeElement(iq.ServiceUnavailableError())
		case router.ErrFailedRemoteConnect:
			s.writeElement(iq.RemoteServerNotFoundError())
		case router.ErrBlockedJID:
			// Destination user is a blocked JID
			if iq.IsGet() || iq.IsSet() {
				s.writeElement(iq.ServiceUnavailableError())
			}
		}
		return
	}
	s.mods.ProcessIQ(iq, s)
}

func (s *inStream) processPresence(presence *xmpp.Presence) {
	if presence.ToJID().IsFullWithUser() {
		s.router.Route(presence)
		return
	}
	replyOnBehalf := s.JID().Matches(presence.ToJID(), jid.MatchesBare)

	// Update presence
	if replyOnBehalf && (presence.IsAvailable() || presence.IsUnavailable()) {
		s.setPresence(presence)

		// Let the whole cluster know that there has been a change in our presence
		s.router.UpdateClusterPresence(presence, s.JID())
	}
	// Deliver presence to roster module
	if r := s.mods.Roster; r != nil {
		r.ProcessPresence(presence)
	}
	// Deliver offline messages
	if replyOnBehalf && presence.IsAvailable() && presence.Priority() >= 0 {
		if off := s.mods.Offline; off != nil {
			off.DeliverOfflineMessages(s)
		}
	}
}

func (s *inStream) processMessage(message *xmpp.Message) {
	msg := message

sendMessage:
	err := s.router.Route(msg)
	switch err {
	case nil:
		break
	case router.ErrResourceNotFound:
		// Treat the stanza as if it were addressed to <node@domain>
		msg, _ = xmpp.NewMessageFromElement(msg, msg.FromJID(), msg.ToJID().ToBareJID())
		goto sendMessage
	case router.ErrNotAuthenticated:
		if off := s.mods.Offline; off != nil {
			off.ArchiveMessage(message)
			return
		}
		fallthrough
	case router.ErrNotExistingAccount, router.ErrBlockedJID:
		s.writeElement(message.ServiceUnavailableError())
	case router.ErrFailedRemoteConnect:
		s.writeElement(message.RemoteServerNotFoundError())
	default:
		log.Error(err)
	}
}

// Runs on it's own goroutine
func (s *inStream) loop() {
	for {
		select {
		case f := <-s.actorCh:
			f()
			if s.getState() == disconnected {
				return
			}
		case resIQ := <-s.iqResultCh:
			if resIQ != nil {
				s.writeElement(resIQ)
			}
		}
	}
}

// Runs on it's own goroutine
func (s *inStream) doRead() {
	elem, sErr := s.sess.Receive()
	if sErr == nil {
		s.actorCh <- func() {
			s.readElement(elem)
		}
	} else {
		s.actorCh <- func() {
			if s.getState() == disconnected {
				return
			}
			s.handleSessionError(sErr)
		}
	}
}

func (s *inStream) handleSessionError(sErr *session.Error) {
	switch err := sErr.UnderlyingErr.(type) {
	case nil:
		s.disconnect(nil)
	case *streamerror.Error:
		s.disconnectWithStreamError(err)
	case *xmpp.StanzaError:
		s.writeStanzaErrorResponse(sErr.Element, err)
	default:
		log.Error(err)
		s.disconnectWithStreamError(streamerror.ErrUndefinedCondition)
	}
}

func (s *inStream) writeStanzaErrorResponse(elem xmpp.XElement, stanzaErr *xmpp.StanzaError) {
	resp := xmpp.NewElementFromElement(elem)
	resp.SetType(xmpp.ErrorType)
	resp.SetFrom(resp.To())
	resp.SetTo(s.JID().String())
	resp.AppendElement(stanzaErr.Element())
	s.writeElement(resp)
}

func (s *inStream) writeElement(elem xmpp.XElement) {
	s.sess.Send(elem)
}

func (s *inStream) readElement(elem xmpp.XElement) {
	if elem != nil {
		s.handleElement(elem)
	}
	if s.getState() != disconnected {
		go s.doRead() // Keep reading...
	}
}

func (s *inStream) disconnect(err error) {
	if s.getState() == disconnected {
		return
	}
	switch err {
	case nil:
		s.disconnectClosingSession(false, true)
	default:
		if stmErr, ok := err.(*streamerror.Error); ok {
			s.disconnectWithStreamError(stmErr)
		} else {
			log.Error(err)
			s.disconnectClosingSession(false, true)
		}
	}
}

func (s *inStream) disconnectWithStreamError(err *streamerror.Error) {
	if s.getState() == connecting {
		s.sess.Open()
	}
	s.writeElement(err.Element())

	unregister := err != streamerror.ErrSystemShutdown
	s.disconnectClosingSession(true, unregister)
}

func (s *inStream) disconnectClosingSession(closeSession, unbind bool) {
	// Stop pinging...
	if p := s.mods.Ping; p != nil {
		p.CancelPing(s)
	}
	// Send 'unavailable' presence when disconnecting
	if presence := s.Presence(); presence != nil && presence.IsAvailable() {
		if r := s.mods.Roster; r != nil {
			r.ProcessPresence(xmpp.NewPresence(s.JID(), s.JID().ToBareJID(), xmpp.UnavailableType))
		}
	}
	if closeSession {
		s.sess.Close()
	}
	// Unregister stream
	if unbind {
		s.router.Unbind(s.JID())
	}
	// Notify disconnection
	if s.cfg.onDisconnect != nil {
		s.cfg.onDisconnect(s)
	}
	s.setState(disconnected)
	s.cfg.transport.Close()
}

func (s *inStream) isBlockedJID(j *jid.JID) bool {
	if j.IsServer() && s.router.IsLocalHost(j.Domain()) {
		return false
	}
	return s.router.IsBlockedJID(j, s.Username())
}

func (s *inStream) restartSession() {
	s.sess = session.New(s.id, &session.Config{
		JID:           s.JID(),
		Transport:     s.cfg.transport,
		MaxStanzaSize: s.cfg.maxStanzaSize,
	}, s.router)
	s.setState(connecting)
}

func (s *inStream) setJID(j *jid.JID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jid = j
}

func (s *inStream) setSecured(secured bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secured = secured
}

func (s *inStream) setAuthenticated(authenticated bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authenticated = authenticated
}

func (s *inStream) isCompressed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.compressed
}

func (s *inStream) setCompressed(compressed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.compressed = compressed
}

func (s *inStream) setPresence(presence *xmpp.Presence) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.presence = presence
}

func (s *inStream) setState(state uint32) {
	atomic.StoreUint32(&s.state, state)
}

func (s *inStream) getState() uint32 {
	return atomic.LoadUint32(&s.state)
}
