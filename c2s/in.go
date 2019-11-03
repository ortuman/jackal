/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"crypto/tls"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/auth"
	"github.com/ortuman/jackal/cluster"
	"github.com/ortuman/jackal/component"
	streamerror "github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/runqueue"
	"github.com/ortuman/jackal/session"
	"github.com/ortuman/jackal/storage"
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
	bound
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
	authenticators []auth.Authenticator
	activeAuth     auth.Authenticator
	runQueue       *runqueue.RunQueue

	mu            sync.RWMutex
	jid           *jid.JID
	secured       bool
	compressed    bool
	authenticated bool
	sessStarted   bool
	presence      *xmpp.Presence

	contextMu sync.RWMutex
	context   map[string]interface{}
}

func newStream(id string, config *streamConfig, mods *module.Modules, comps *component.Components, router *router.Router) stream.C2S {
	s := &inStream{
		cfg:      config,
		router:   router,
		mods:     mods,
		comps:    comps,
		id:       id,
		context:  make(map[string]interface{}),
		runQueue: runqueue.New(id),
	}

	// initialize stream context
	secured := !(config.transport.Type() == transport.Socket)
	s.setSecured(secured)
	s.setJID(&jid.JID{})

	// initialize authenticators
	s.initializeAuthenticators()

	// start c2s session
	s.restartSession()

	if config.connectTimeout > 0 {
		s.connectTm = time.AfterFunc(config.connectTimeout, s.connectTimeout)
	}
	go s.doRead() // start reading...

	return s
}

// ID returns stream identifier.
func (s *inStream) ID() string {
	return s.id
}

// Context returns a copy of the stream associated context.
func (s *inStream) Context() map[string]interface{} {
	m := make(map[string]interface{})
	s.contextMu.RLock()
	for k, v := range s.context {
		m[k] = v
	}
	s.contextMu.RUnlock()
	return m
}

// SetString associates a string context value to a key.
func (s *inStream) SetString(key string, value string) {
	s.setContextValue(key, value)
}

// GetString returns the context value associated with the key as a string.
func (s *inStream) GetString(key string) string {
	var ret string
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if s, ok := s.context[key].(string); ok {
		ret = s
	}
	return ret
}

// SetInt associates an integer context value to a key.
func (s *inStream) SetInt(key string, value int) {
	s.setContextValue(key, value)
}

// GetInt returns the context value associated with the key as an integer.
func (s *inStream) GetInt(key string) int {
	var ret int
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if i, ok := s.context[key].(int); ok {
		ret = i
	}
	return ret
}

// SetFloat associates a float context value to a key.
func (s *inStream) SetFloat(key string, value float64) {
	s.setContextValue(key, value)
}

// GetFloat returns the context value associated with the key as a float64.
func (s *inStream) GetFloat(key string) float64 {
	var ret float64
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if f, ok := s.context[key].(float64); ok {
		ret = f
	}
	return ret
}

// SetBool associates a boolean context value to a key.
func (s *inStream) SetBool(key string, value bool) {
	s.setContextValue(key, value)
}

// GetBool returns the context value associated with the key as a boolean.
func (s *inStream) GetBool(key string) bool {
	var ret bool
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if b, ok := s.context[key].(bool); ok {
		ret = b
	}
	return ret
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

// SendElement writes an XMPP element to the stream.
func (s *inStream) SendElement(elem xmpp.XElement) {
	if s.getState() == disconnected {
		return
	}
	s.runQueue.Run(func() { s.writeElement(elem) })
}

// Disconnect disconnects remote peer by closing the underlying TCP socket connection.
func (s *inStream) Disconnect(err error) {
	if s.getState() == disconnected {
		return
	}
	waitCh := make(chan struct{})
	s.runQueue.Run(func() {
		s.disconnect(err)
		close(waitCh)
	})
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

		case "scram_sha_512":
			authenticators = append(authenticators, auth.NewScram(s, tr, auth.ScramSHA512, false))
			authenticators = append(authenticators, auth.NewScram(s, tr, auth.ScramSHA512, true))
		}
	}
	s.authenticators = authenticators
}

func (s *inStream) connectTimeout() {
	s.runQueue.Run(func() { s.disconnect(streamerror.ErrConnectionTimeout) })
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
	case bound:
		s.handleBound(elem)
	}
}

func (s *inStream) handleConnecting(elem xmpp.XElement) {
	// cancel connection timeout timer
	if s.connectTm != nil {
		s.connectTm.Stop()
		s.connectTm = nil
	}
	// assign stream domain if not set yet
	if len(s.Domain()) == 0 {
		j, _ := jid.New("", elem.To(), "", true)
		s.setJID(j)
	}

	// open stream session
	s.sess.SetJID(s.JID())

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
	_ = s.sess.Open(features)
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

	// attach SASL mechanisms
	shouldOfferSASL := !isSocketTr || (isSocketTr && s.IsSecured())

	if shouldOfferSASL && len(s.authenticators) > 0 {
		mechanisms := xmpp.NewElementName("mechanisms")
		mechanisms.SetNamespace(saslNamespace)
		for _, ath := range s.authenticators {
			mechanism := xmpp.NewElementName("mechanism")
			mechanism.SetText(ath.Mechanism())
			mechanisms.AppendElement(mechanism)
		}
		features = append(features, mechanisms)
	}

	// allow In-band registration over encrypted stream only
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

	// attach compression feature
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

	// [rfc6121] offer session feature for backward compatibility
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
				reg.ProcessIQWithStream(iq, s)
			} else {
				// channel isn't safe enough to enable a password change
				s.writeElement(iq.NotAuthorizedError())
			}
			return

		} else if iq.Elements().ChildNamespace("query", "jabber:iq:auth") != nil {
			// don't allow non-SASL authentication
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
	ath := s.activeAuth
	_ = s.continueAuthentication(elem, ath)
	if ath.Authenticated() {
		s.finishAuthentication(ath.Username())
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
		}

	default:
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *inStream) handleBound(elem xmpp.XElement) {
	// reset ping timer deadline
	if p := s.mods.Ping; p != nil {
		p.SchedulePing(s)
	}
	stanza, ok := elem.(xmpp.Stanza)
	if !ok {
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
		return
	}
	// handle session IQ
	if iq, ok := stanza.(*xmpp.IQ); ok && iq.IsSet() {
		if iq.Elements().ChildNamespace("session", sessionNamespace) != nil {
			if !s.isSessionStarted() {
				s.setSessionStarted(true)
				s.writeElement(iq.ResultIQ())
			} else {
				s.writeElement(iq.NotAllowedError())
			}
			return
		}
	}
	if comp := s.comps.Get(stanza.ToJID().Domain()); comp != nil { // component stanza?
		switch stanza := stanza.(type) {
		case *xmpp.IQ:
			if di := s.mods.DiscoInfo; di != nil && di.MatchesIQ(stanza) {
				di.ProcessIQ(stanza)
				return
			}
			break
		}
		comp.ProcessStanza(stanza, s)
		return
	}
	s.processStanza(stanza)
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
	s.setSecured(true)
	s.writeElement(xmpp.NewElementNamespace("proceed", tlsNamespace))

	s.cfg.transport.StartTLS(&tls.Config{Certificates: s.router.Certificates()}, false)

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
	for _, authenticator := range s.authenticators {
		if authenticator.Mechanism() == mechanism {
			if err := s.continueAuthentication(elem, authenticator); err != nil {
				return
			}
			if authenticator.Authenticated() {
				s.finishAuthentication(authenticator.Username())
			} else {
				s.activeAuth = authenticator
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
	// try binding...
	var stm stream.C2S
	streams := s.router.UserStreams(s.JID().Node())
	for _, s := range streams {
		if s.Resource() == resource {
			stm = s
		}
	}
	if stm != nil {
		switch s.cfg.resourceConflict {
		case Override:
			// override the resource with a server-generated resourcepart...
			resource = uuid.New()
		case Replace:
			// terminate the session of the currently connected client...
			stm.Disconnect(streamerror.ErrResourceConstraint)
		default:
			// disallow resource binding attempt...
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

	s.mu.Lock()
	s.presence = xmpp.NewPresence(userJID, userJID, xmpp.UnavailableType)
	s.mu.Unlock()

	s.router.Bind(s)

	//...notify successful binding
	result := xmpp.NewIQType(iq.ID(), xmpp.ResultType)
	result.SetNamespace(iq.Namespace())

	boundElem := xmpp.NewElementNamespace("bind", bindNamespace)
	j := xmpp.NewElementName("jid")
	j.SetText(s.Username() + "@" + s.Domain() + "/" + s.Resource())
	boundElem.AppendElement(j)
	result.AppendElement(boundElem)

	s.setState(bound)
	s.writeElement(result)

	// start pinging...
	if p := s.mods.Ping; p != nil {
		p.SchedulePing(s)
	}
}

func (s *inStream) processStanza(elem xmpp.Stanza) {
	toJID := elem.ToJID()
	if s.isBlockedJID(toJID) { // blocked JID?
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
			// destination user is a blocked JID
			if iq.IsGet() || iq.IsSet() {
				s.writeElement(iq.ServiceUnavailableError())
			}
		}
		return
	}
	// process capabilities result
	caps := iq.Elements().ChildNamespace("query", discoInfoNamespace)
	if caps != nil && iq.IsResult() {
		s.processCapabilitiesResponse(caps)
		return
	}
	s.mods.ProcessIQ(iq)
}

func (s *inStream) processCapabilitiesResponse(query xmpp.XElement) {
	var node, ver string

	nodeStr := query.Attributes().Get("node")
	ss := strings.Split(nodeStr, "#")
	if len(ss) != 2 {
		log.Warnf("wrong node format: %s", nodeStr)
		return
	}
	node = ss[0]
	ver = ss[1]

	// retrieve and store features
	log.Infof("storing capabilities... node: %s, ver: %s", node, ver)

	var features []string
	featureElems := query.Elements().Children("feature")
	for _, featureElem := range featureElems {
		features = append(features, featureElem.Attributes().Get("var"))
	}
	if err := storage.InsertCapabilities(node, ver, &model.Capabilities{Features: features}); err != nil {
		log.Warnf("%v", err)
		return
	}
}

func (s *inStream) processPresence(presence *xmpp.Presence) {
	if presence.ToJID().IsFullWithUser() {
		_ = s.router.Route(presence)
		return
	}
	replyOnBehalf := s.JID().Matches(presence.ToJID(), jid.MatchesBare)

	// update presence
	if replyOnBehalf && (presence.IsAvailable() || presence.IsUnavailable()) {
		s.setPresence(presence)
	}
	// deliver presence to roster module
	if r := s.mods.Roster; r != nil {
		r.ProcessPresence(presence)
	}
	// deliver offline messages
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
		// treat the stanza as if it were addressed to <node@domain>
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
func (s *inStream) doRead() {
	elem, sErr := s.sess.Receive()
	if sErr == nil {
		s.runQueue.Run(func() { s.readElement(elem) })
	} else {
		s.runQueue.Run(func() {
			if s.getState() == disconnected {
				return
			}
			s.handleSessionError(sErr)
		})
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
		_ = s.sess.Open(nil)
	}
	s.writeElement(err.Element())

	unregister := err != streamerror.ErrSystemShutdown
	s.disconnectClosingSession(true, unregister)
}

func (s *inStream) disconnectClosingSession(closeSession, unbind bool) {
	// stop pinging...
	if p := s.mods.Ping; p != nil {
		p.CancelPing(s)
	}
	// send 'unavailable' presence when disconnecting
	if presence := s.Presence(); presence != nil && presence.IsAvailable() {
		if r := s.mods.Roster; r != nil {
			r.ProcessPresence(xmpp.NewPresence(s.JID(), s.JID().ToBareJID(), xmpp.UnavailableType))
		}
	}
	if closeSession {
		_ = s.sess.Close()
	}
	// unregister stream
	if unbind {
		s.router.Unbind(s.JID())
	}
	// notify disconnection
	if s.cfg.onDisconnect != nil {
		s.cfg.onDisconnect(s)
	}
	s.setState(disconnected)
	_ = s.cfg.transport.Close()

	s.runQueue.Stop(nil) // stop processing messages
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

func (s *inStream) setContextValue(key string, value interface{}) {
	s.contextMu.Lock()
	s.context[key] = value
	s.contextMu.Unlock()

	// notify the whole roster about the context update.
	if c := s.router.Cluster(); c != nil {
		c.BroadcastMessage(&cluster.Message{
			Type: cluster.MsgUpdateContext,
			Node: c.LocalNode(),
			Payloads: []cluster.MessagePayload{{
				JID:     s.JID(),
				Context: map[string]interface{}{key: value},
			}},
		})
	}
}

func (s *inStream) setPresence(presence *xmpp.Presence) {
	s.mu.Lock()
	s.presence = presence
	s.mu.Unlock()

	// request entity capabilities if needed
	if caps := presence.Capabilities(); caps != nil {
		ok, err := storage.HasCapabilities(caps.Node, caps.Ver)
		switch err {
		case nil:
			if !ok {
				srvJID, _ := jid.NewWithString(s.Domain(), true)

				iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
				iq.SetFromJID(srvJID)
				iq.SetToJID(s.JID())

				query := xmpp.NewElementNamespace("query", discoInfoNamespace)
				query.SetAttribute("node", caps.Node+"#"+caps.Ver)
				iq.AppendElement(query)

				log.Infof("requesting capabilities... node: %s, ver: %s", caps.Node, caps.Ver)
				s.writeElement(iq)
			}
		default:
			log.Warnf("%v", err)
		}
	}
	// notify the whole roster about the presence update.
	if c := s.router.Cluster(); c != nil {
		c.BroadcastMessage(&cluster.Message{
			Type: cluster.MsgUpdatePresence,
			Node: c.LocalNode(),
			Payloads: []cluster.MessagePayload{{
				JID:    s.jid,
				Stanza: presence,
			}},
		})
	}
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

func (s *inStream) isSessionStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessStarted
}

func (s *inStream) setSessionStarted(sessStarted bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessStarted = sessStarted
}

func (s *inStream) setState(state uint32) {
	atomic.StoreUint32(&s.state, state)
}

func (s *inStream) getState() uint32 {
	return atomic.LoadUint32(&s.state)
}
