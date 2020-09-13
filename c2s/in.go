/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"context"
	"crypto/tls"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/ortuman/jackal/auth"
	"github.com/ortuman/jackal/component"
	streamerror "github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/session"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/transport/compress"
	"github.com/ortuman/jackal/util/runqueue"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
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
	router         router.Router
	userRep        repository.User
	blockListRep   repository.BlockList
	mods           *module.Modules
	comps          *component.Components
	sess           *session.Session
	tr             transport.Transport
	mu             sync.RWMutex
	id             string
	connectTm      *time.Timer
	readTimeoutTm  *time.Timer
	state          uint32
	authenticators []auth.Authenticator
	activeAuth     auth.Authenticator
	runQueue       *runqueue.RunQueue
	jid            *jid.JID
	secured        bool
	compressed     bool
	authenticated  bool
	sessStarted    bool
	presence       *xmpp.Presence
	ctx            context.Context
	ctxCancelFn    context.CancelFunc
}

func newStream(id string, config *streamConfig, tr transport.Transport, mods *module.Modules, comps *component.Components, router router.Router, userRep repository.User, blockListRep repository.BlockList) stream.C2S {
	ctx, ctxCancelFn := context.WithCancel(context.Background())
	s := &inStream{
		cfg:          config,
		tr:           tr,
		router:       router,
		userRep:      userRep,
		blockListRep: blockListRep,
		mods:         mods,
		comps:        comps,
		id:           id,
		runQueue:     runqueue.New(id),
		ctx:          ctx,
		ctxCancelFn:  ctxCancelFn,
	}

	// initialize stream context
	secured := !(tr.Type() == transport.Socket)
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

func (s *inStream) Context() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ctx
}

func (s *inStream) Value(key interface{}) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ctx.Value(key)
}

func (s *inStream) SetValue(key, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = context.WithValue(s.ctx, key, value)
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
func (s *inStream) SendElement(ctx context.Context, elem xmpp.XElement) {
	if s.getState() == disconnected {
		return
	}
	s.runQueue.Run(func() { s.writeElement(ctx, elem) })
}

// Disconnect disconnects remote peer by closing the underlying TCP socket connection.
func (s *inStream) Disconnect(ctx context.Context, err error) {
	if s.getState() == disconnected {
		return
	}
	waitCh := make(chan struct{})
	s.runQueue.Run(func() {
		s.disconnect(ctx, err)
		close(waitCh)
	})
	<-waitCh
}

func (s *inStream) initializeAuthenticators() {
	tr := s.tr
	hasChannelBinding := len(tr.ChannelBindingBytes(transport.TLSUnique)) > 0
	var authenticators []auth.Authenticator
	for _, a := range s.cfg.sasl {
		switch a {
		case "plain":
			authenticators = append(authenticators, auth.NewPlain(s, s.userRep))

		case "scram_sha_1":
			authenticators = append(authenticators, auth.NewScram(s, tr, auth.ScramSHA1, false, s.userRep))
			if hasChannelBinding {
				authenticators = append(authenticators, auth.NewScram(s, tr, auth.ScramSHA1, true, s.userRep))
			}

		case "scram_sha_256":
			authenticators = append(authenticators, auth.NewScram(s, tr, auth.ScramSHA256, false, s.userRep))
			if hasChannelBinding {
				authenticators = append(authenticators, auth.NewScram(s, tr, auth.ScramSHA256, true, s.userRep))
			}
		}
	}
	s.authenticators = authenticators
}

func (s *inStream) connectTimeout() {
	s.runQueue.Run(func() {
		ctx, _ := context.WithTimeout(context.Background(), s.cfg.timeout)
		s.disconnect(ctx, streamerror.ErrConnectionTimeout)
	})
}

func (s *inStream) handleElement(ctx context.Context, elem xmpp.XElement) {
	switch s.getState() {
	case connecting:
		s.handleConnecting(ctx, elem)
	case connected:
		s.handleConnected(ctx, elem)
	case authenticated:
		s.handleAuthenticated(ctx, elem)
	case authenticating:
		s.handleAuthenticating(ctx, elem)
	case bound:
		s.handleBound(ctx, elem)
	}
}

func (s *inStream) handleConnecting(ctx context.Context, elem xmpp.XElement) {
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
	_ = s.sess.Open(ctx, features)
}

func (s *inStream) unauthenticatedFeatures() []xmpp.XElement {
	var features []xmpp.XElement

	isSocketTr := s.tr.Type() == transport.Socket

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

	isSocketTr := s.tr.Type() == transport.Socket

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

func (s *inStream) handleConnected(ctx context.Context, elem xmpp.XElement) {
	switch elem.Name() {
	case "starttls":
		s.proceedStartTLS(ctx, elem)

	case "auth":
		s.startAuthentication(ctx, elem)

	case "iq":
		iq := elem.(*xmpp.IQ)
		if reg := s.mods.Register; reg != nil && reg.MatchesIQ(iq) {
			if s.IsSecured() {
				reg.ProcessIQWithStream(ctx, iq, s)
			} else {
				// channel isn't safe enough to enable a password change
				s.writeElement(ctx, iq.NotAuthorizedError())
			}
			return

		} else if iq.Elements().ChildNamespace("query", "jabber:iq:auth") != nil {
			// don't allow non-SASL authentication
			s.writeElement(ctx, iq.ServiceUnavailableError())
			return
		}
		fallthrough

	case "message", "presence":
		s.disconnectWithStreamError(ctx, streamerror.ErrNotAuthorized)

	default:
		s.disconnectWithStreamError(ctx, streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *inStream) handleAuthenticating(ctx context.Context, elem xmpp.XElement) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(ctx, streamerror.ErrInvalidNamespace)
		return
	}
	ath := s.activeAuth
	_ = s.continueAuthentication(ctx, elem, ath)
	if ath.Authenticated() {
		s.finishAuthentication(ctx, ath.Username())
	}
}

func (s *inStream) handleAuthenticated(ctx context.Context, elem xmpp.XElement) {
	switch elem.Name() {
	case "compress":
		if elem.Namespace() != compressProtocolNamespace {
			s.disconnectWithStreamError(ctx, streamerror.ErrUnsupportedStanzaType)
			return
		}
		s.compress(ctx, elem)

	case "iq":
		iq := elem.(*xmpp.IQ)
		if len(s.JID().Resource()) == 0 { // Expecting bind
			s.bindResource(ctx, iq)
		}

	default:
		s.disconnectWithStreamError(ctx, streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *inStream) handleBound(ctx context.Context, elem xmpp.XElement) {
	// reset ping timer deadline
	if p := s.mods.Ping; p != nil {
		p.SchedulePing(s)
	}
	stanza, ok := elem.(xmpp.Stanza)
	if !ok {
		s.disconnectWithStreamError(ctx, streamerror.ErrUnsupportedStanzaType)
		return
	}
	// handle session IQ
	if iq, ok := stanza.(*xmpp.IQ); ok && iq.IsSet() {
		if iq.Elements().ChildNamespace("session", sessionNamespace) != nil {
			if !s.isSessionStarted() {
				s.setSessionStarted(true)
				s.writeElement(ctx, iq.ResultIQ())
			} else {
				s.writeElement(ctx, iq.NotAllowedError())
			}
			return
		}
	}
	if comp := s.comps.Get(stanza.ToJID().Domain()); comp != nil { // component stanza?
		switch stanza := stanza.(type) {
		case *xmpp.IQ:
			if di := s.mods.DiscoInfo; di != nil && di.MatchesIQ(stanza) {
				di.ProcessIQ(ctx, stanza)
				return
			}
			break
		}
		comp.ProcessStanza(ctx, stanza, s)
		return
	}
	s.processStanza(ctx, stanza)
}

func (s *inStream) proceedStartTLS(ctx context.Context, elem xmpp.XElement) {
	if s.IsSecured() {
		s.disconnectWithStreamError(ctx, streamerror.ErrNotAuthorized)
		return
	}
	if len(elem.Namespace()) > 0 && elem.Namespace() != tlsNamespace {
		s.disconnectWithStreamError(ctx, streamerror.ErrInvalidNamespace)
		return
	}
	s.setSecured(true)
	s.writeElement(ctx, xmpp.NewElementNamespace("proceed", tlsNamespace))

	s.tr.StartTLS(&tls.Config{Certificates: s.router.Hosts().Certificates()}, false)

	log.Infof("secured stream... id: %s", s.id)
	s.restartSession()
}

func (s *inStream) compress(ctx context.Context, elem xmpp.XElement) {
	if s.isCompressed() {
		s.disconnectWithStreamError(ctx, streamerror.ErrUnsupportedStanzaType)
		return
	}
	method := elem.Elements().Child("method")
	if method == nil || len(method.Text()) == 0 {
		failure := xmpp.NewElementNamespace("failure", compressProtocolNamespace)
		failure.AppendElement(xmpp.NewElementName("setup-failed"))
		s.writeElement(ctx, failure)
		return
	}
	if method.Text() != "zlib" {
		failure := xmpp.NewElementNamespace("failure", compressProtocolNamespace)
		failure.AppendElement(xmpp.NewElementName("unsupported-method"))
		s.writeElement(ctx, failure)
		return
	}
	s.writeElement(ctx, xmpp.NewElementNamespace("compressed", compressProtocolNamespace))

	s.tr.EnableCompression(s.cfg.compression.Level)
	s.setCompressed(true)

	log.Infof("compressed stream... id: %s", s.id)

	s.restartSession()
}

func (s *inStream) startAuthentication(ctx context.Context, elem xmpp.XElement) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(ctx, streamerror.ErrInvalidNamespace)
		return
	}
	mechanism := elem.Attributes().Get("mechanism")
	for _, authenticator := range s.authenticators {
		if authenticator.Mechanism() == mechanism {
			if err := s.continueAuthentication(ctx, elem, authenticator); err != nil {
				return
			}
			if authenticator.Authenticated() {
				s.finishAuthentication(ctx, authenticator.Username())
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
	s.writeElement(ctx, failure)
}

func (s *inStream) continueAuthentication(ctx context.Context, elem xmpp.XElement, authr auth.Authenticator) error {
	err := authr.ProcessElement(ctx, elem)
	if saslErr, ok := err.(*auth.SASLError); ok {
		s.failAuthentication(ctx, saslErr.Element())
	} else if err != nil {
		log.Error(err)
		s.failAuthentication(ctx, auth.ErrSASLTemporaryAuthFailure.(*auth.SASLError).Element())
	}
	return err
}

func (s *inStream) finishAuthentication(_ context.Context, username string) {
	if s.activeAuth != nil {
		s.activeAuth.Reset()
		s.activeAuth = nil
	}
	j, _ := jid.New(username, s.Domain(), "", true)
	s.setJID(j)
	s.setAuthenticated(true)

	s.restartSession()
}

func (s *inStream) failAuthentication(ctx context.Context, elem xmpp.XElement) {
	failure := xmpp.NewElementNamespace("failure", saslNamespace)
	failure.AppendElement(elem)
	s.writeElement(ctx, failure)

	if s.activeAuth != nil {
		s.activeAuth.Reset()
		s.activeAuth = nil
	}
	s.setState(connected)
}

func (s *inStream) bindResource(ctx context.Context, iq *xmpp.IQ) {
	bind := iq.Elements().ChildNamespace("bind", bindNamespace)
	if bind == nil {
		s.writeElement(ctx, iq.NotAllowedError())
		return
	}
	var resource string
	if resourceElem := bind.Elements().Child("resource"); resourceElem != nil {
		resource = resourceElem.Text()
	} else {
		resource = uuid.New().String()
	}
	// try binding...
	var stm stream.C2S
	streams := s.router.LocalStreams(s.JID().Node())
	for _, s := range streams {
		if s.Resource() == resource {
			stm = s
		}
	}
	if stm != nil {
		switch s.cfg.resourceConflict {
		case Override:
			// override the resource with a server-generated resourcepart...
			resource = uuid.New().String()
		case Replace:
			// terminate the session of the currently connected client...
			stm.Disconnect(ctx, streamerror.ErrResourceConstraint)
		default:
			// disallow resource binding attempt...
			s.writeElement(ctx, iq.ConflictError())
			return
		}
	}
	userJID, err := jid.New(s.Username(), s.Domain(), resource, false)
	if err != nil {
		s.writeElement(ctx, iq.BadRequestError())
		return
	}
	s.setJID(userJID)
	s.sess.SetJID(userJID)

	s.mu.Lock()
	s.presence = xmpp.NewPresence(userJID, userJID, xmpp.UnavailableType)
	s.mu.Unlock()

	s.router.Bind(ctx, s)

	//...notify successful binding
	result := xmpp.NewIQType(iq.ID(), xmpp.ResultType)
	result.SetNamespace(iq.Namespace())

	boundElem := xmpp.NewElementNamespace("bind", bindNamespace)
	j := xmpp.NewElementName("jid")
	j.SetText(s.Username() + "@" + s.Domain() + "/" + s.Resource())
	boundElem.AppendElement(j)
	result.AppendElement(boundElem)

	s.setState(bound)
	s.writeElement(ctx, result)

	// start pinging...
	if p := s.mods.Ping; p != nil {
		p.SchedulePing(s)
	}
}

func (s *inStream) processStanza(ctx context.Context, elem xmpp.Stanza) {
	toJID := elem.ToJID()
	if s.isBlockedJID(ctx, toJID) { // blocked JID?
		blocked := xmpp.NewElementNamespace("blocked", blockedErrorNamespace)
		resp := xmpp.NewErrorStanzaFromStanza(elem, xmpp.ErrNotAcceptable, []xmpp.XElement{blocked})
		s.writeElement(ctx, resp)
		return
	}
	switch stanza := elem.(type) {
	case *xmpp.Presence:
		s.processPresence(ctx, stanza)
	case *xmpp.IQ:
		s.processIQ(ctx, stanza)
	case *xmpp.Message:
		s.processMessage(ctx, stanza)
	}
}

func (s *inStream) processIQ(ctx context.Context, iq *xmpp.IQ) {
	toJID := iq.ToJID()
	replyOnBehalf := !toJID.IsFullWithUser() && (s.router.Hosts().IsLocalHost(toJID.Domain())) ||
		s.router.Hosts().IsConferenceHost(toJID.Domain())
	if !replyOnBehalf {
		switch s.router.Route(ctx, iq) {
		case router.ErrResourceNotFound:
			s.writeElement(ctx, iq.ServiceUnavailableError())
		case router.ErrFailedRemoteConnect:
			s.writeElement(ctx, iq.RemoteServerNotFoundError())
		case router.ErrBlockedJID:
			// destination user is a blocked JID
			if iq.IsGet() || iq.IsSet() {
				s.writeElement(ctx, iq.ServiceUnavailableError())
			}
		}
		return
	}
	s.mods.ProcessIQ(ctx, iq)
}

func (s *inStream) processPresence(ctx context.Context, presence *xmpp.Presence) {
	// is the presence stanza directed to the conference service?
	if s.router.Hosts().IsConferenceHost(presence.ToJID().Domain()) {
		s.mods.Muc.ProcessPresence(ctx, presence)
		return
	}

	if presence.ToJID().IsFullWithUser() {
		_ = s.router.Route(ctx, presence)
		return
	}
	replyOnBehalf := s.JID().MatchesWithOptions(presence.ToJID(), jid.MatchesBare)

	// update presence
	if replyOnBehalf && (presence.IsAvailable() || presence.IsUnavailable()) {
		s.setPresence(presence)
	}
	// process presence
	if r := s.mods.Roster; r != nil {
		r.ProcessPresence(ctx, presence)
	}

	// deliver offline messages
	if replyOnBehalf && presence.IsAvailable() && presence.Priority() >= 0 {
		if off := s.mods.Offline; off != nil {
			off.DeliverOfflineMessages(ctx, s)
		}
	}
}

func (s *inStream) processMessage(ctx context.Context, message *xmpp.Message) {
	msg := message

sendMessage:
	err := s.router.Route(ctx, msg)
	switch err {
	case nil:
		break
	case router.ErrResourceNotFound:
		// treat the stanza as if it were addressed to <node@domain>
		msg, _ = xmpp.NewMessageFromElement(msg, msg.FromJID(), msg.ToJID().ToBareJID())
		goto sendMessage
	case router.ErrNotAuthenticated:
		if off := s.mods.Offline; off != nil {
			off.ArchiveMessage(ctx, message)
			return
		}
		fallthrough
	case router.ErrNotExistingAccount, router.ErrBlockedJID:
		s.writeElement(ctx, message.ServiceUnavailableError())
	case router.ErrFailedRemoteConnect:
		s.writeElement(ctx, message.RemoteServerNotFoundError())
	default:
		log.Error(err)
	}
}

// Runs on it's own goroutine
func (s *inStream) doRead() {
	s.scheduleReadTimeout()
	elem, sErr := s.sess.Receive()
	s.cancelReadTimeout()

	ctx, _ := context.WithTimeout(context.Background(), s.cfg.timeout)
	if sErr == nil {
		s.runQueue.Run(func() { s.readElement(ctx, elem) })
	} else {
		s.runQueue.Run(func() {
			if s.getState() == disconnected {
				return
			}
			s.handleSessionError(ctx, sErr)
		})
	}
}

func (s *inStream) handleSessionError(ctx context.Context, sErr *session.Error) {
	switch err := sErr.UnderlyingErr.(type) {
	case nil:
		s.disconnect(ctx, nil)
	case *streamerror.Error:
		s.disconnectWithStreamError(ctx, err)
	case *xmpp.StanzaError:
		s.writeStanzaErrorResponse(ctx, sErr.Element, err)
	default:
		log.Error(err)
		s.disconnectWithStreamError(ctx, streamerror.ErrUndefinedCondition)
	}
}

func (s *inStream) writeStanzaErrorResponse(ctx context.Context, elem xmpp.XElement, stanzaErr *xmpp.StanzaError) {
	resp := xmpp.NewElementFromElement(elem)
	resp.SetType(xmpp.ErrorType)
	resp.SetFrom(resp.To())
	resp.SetTo(s.JID().String())
	resp.AppendElement(stanzaErr.Element())
	s.writeElement(ctx, resp)
}

func (s *inStream) writeElement(ctx context.Context, elem xmpp.XElement) {
	if err := s.sess.Send(ctx, elem); err != nil {
		log.Error(err)
	}
}

func (s *inStream) readElement(ctx context.Context, elem xmpp.XElement) {
	if elem != nil {
		s.handleElement(ctx, elem)
	}
	if s.getState() != disconnected {
		go s.doRead() // keep reading...
	}
}

func (s *inStream) disconnect(ctx context.Context, err error) {
	if s.getState() == disconnected {
		return
	}
	switch err {
	case nil:
		s.disconnectClosingSession(ctx, false, true)
	default:
		if stmErr, ok := err.(*streamerror.Error); ok {
			s.disconnectWithStreamError(ctx, stmErr)
		} else {
			log.Error(err)
			s.disconnectClosingSession(ctx, false, true)
		}
	}
}

func (s *inStream) disconnectWithStreamError(ctx context.Context, err *streamerror.Error) {
	if s.getState() == connecting {
		_ = s.sess.Open(ctx, nil)
	}
	s.writeElement(ctx, err.Element())

	unregister := err != streamerror.ErrSystemShutdown
	s.disconnectClosingSession(ctx, true, unregister)
}

func (s *inStream) disconnectClosingSession(ctx context.Context, closeSession, unbind bool) {
	// stop pinging...
	if p := s.mods.Ping; p != nil {
		p.CancelPing(s)
	}
	// send 'unavailable' presence when disconnecting
	if presence := s.Presence(); presence != nil && presence.IsAvailable() {
		if r := s.mods.Roster; r != nil {
			r.ProcessPresence(ctx, xmpp.NewPresence(s.JID(), s.JID().ToBareJID(), xmpp.UnavailableType))
		}
	}
	if closeSession {
		_ = s.sess.Close(ctx)
	}
	// unregister stream
	if unbind {
		s.router.Unbind(ctx, s.JID())
	}
	s.ctxCancelFn()

	// notify disconnection
	if s.cfg.onDisconnect != nil {
		s.cfg.onDisconnect(s)
	}
	s.setState(disconnected)
	_ = s.tr.Close()

	s.runQueue.Stop(nil) // stop processing messages
}

func (s *inStream) isBlockedJID(ctx context.Context, j *jid.JID) bool {
	blockList, err := s.blockListRep.FetchBlockListItems(ctx, s.Username())
	if err != nil {
		log.Error(err)
		return false
	}
	if len(blockList) == 0 {
		return false
	}
	blockListJIDs := make([]jid.JID, len(blockList))
	for i, listItem := range blockList {
		j, _ := jid.NewWithString(listItem.JID, true)
		blockListJIDs[i] = *j
	}
	for _, blockedJID := range blockListJIDs {
		if blockedJID.Matches(j) {
			return true
		}
	}
	return false
}

func (s *inStream) restartSession() {
	s.sess = session.New(s.id, &session.Config{
		JID:           s.JID(),
		MaxStanzaSize: s.cfg.maxStanzaSize,
	}, s.tr, s.router.Hosts())
	s.setState(connecting)
}

func (s *inStream) setPresence(presence *xmpp.Presence) {
	s.mu.Lock()
	s.presence = presence
	s.mu.Unlock()
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

func (s *inStream) scheduleReadTimeout() {
	s.mu.Lock()
	s.readTimeoutTm = time.AfterFunc(s.cfg.keepAlive, s.readTimeout)
	s.mu.Unlock()
}

func (s *inStream) cancelReadTimeout() {
	s.mu.Lock()
	s.readTimeoutTm.Stop()
	s.mu.Unlock()
}

func (s *inStream) readTimeout() {
	s.runQueue.Run(func() {
		ctx, _ := context.WithTimeout(context.Background(), s.cfg.timeout)
		s.disconnect(ctx, streamerror.ErrConnectionTimeout)
	})
}

func (s *inStream) setState(state uint32) {
	atomic.StoreUint32(&s.state, state)
}

func (s *inStream) getState() uint32 {
	return atomic.LoadUint32(&s.state)
}
