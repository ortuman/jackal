/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"crypto/tls"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/session"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	inConnecting uint32 = iota
	inConnected
	inDisconnected
)

type inStream struct {
	id            string
	cfg           *streamConfig
	router        *router.Router
	mods          *module.Modules
	localDomain   string
	remoteDomain  string
	state         uint32
	connectTm     *time.Timer
	sess          *session.Session
	secured       uint32
	authenticated uint32
	actorCh       chan func()
}

func newInStream(config *streamConfig, mods *module.Modules, router *router.Router) *inStream {
	s := &inStream{
		id:      nextInID(),
		cfg:     config,
		router:  router,
		mods:    mods,
		actorCh: make(chan func(), streamMailboxSize),
	}
	// start s2s in session
	s.restartSession()

	if config.connectTimeout > 0 {
		s.connectTm = time.AfterFunc(config.connectTimeout, s.connectTimeout)
	}
	go s.loop()
	go s.doRead() // start reading transport...
	return s
}

func (s *inStream) ID() string {
	return s.id
}

func (s *inStream) Disconnect(err error) {
	if s.getState() == inDisconnected {
		return
	}
	waitCh := make(chan struct{})
	s.actorCh <- func() {
		s.disconnect(err)
		close(waitCh)
	}
	<-waitCh
}

func (s *inStream) connectTimeout() {
	s.actorCh <- func() { s.disconnect(streamerror.ErrConnectionTimeout) }
}

// runs on its own goroutine
func (s *inStream) loop() {
	for {
		f := <-s.actorCh
		f()
		if s.getState() == inDisconnected {
			return
		}
	}
}

// runs on its own goroutine
func (s *inStream) doRead() {
	if elem, sErr := s.sess.Receive(); sErr == nil {
		s.actorCh <- func() {
			s.readElement(elem)
		}
	} else {
		s.actorCh <- func() {
			if s.getState() == inDisconnected {
				return // already disconnected...
			}
			s.handleSessionError(sErr)
		}
	}
}

func (s *inStream) handleElement(elem xmpp.XElement) {
	switch s.getState() {
	case inConnecting:
		s.handleConnecting(elem)
	case inConnected:
		s.handleConnected(elem)
	}
}

func (s *inStream) handleConnecting(elem xmpp.XElement) {
	// cancel connection timeout timer
	if s.connectTm != nil {
		s.connectTm.Stop()
		s.connectTm = nil
	}
	// assign domain pair
	s.localDomain = elem.To()
	s.remoteDomain = elem.From()

	// open stream session
	s.sess.SetRemoteDomain(s.remoteDomain)

	j, _ := jid.New("", s.localDomain, "", true)
	s.sess.SetJID(j)

	s.sess.Open()

	features := xmpp.NewElementName("stream:features")
	features.SetAttribute("xmlns:stream", streamNamespace)
	features.SetAttribute("version", "1.0")

	if !s.isSecured() {
		starttls := xmpp.NewElementNamespace("starttls", tlsNamespace)
		starttls.AppendElement(xmpp.NewElementName("required"))
		features.AppendElement(starttls)
		s.setState(inConnected)
		s.writeElement(features)
		return
	}
	if !s.isAuthenticated() {
		// offer external authentication
		mechanisms := xmpp.NewElementName("mechanisms")
		mechanisms.SetNamespace(saslNamespace)
		extMech := xmpp.NewElementName("mechanism")
		extMech.SetText("EXTERNAL")
		mechanisms.AppendElement(extMech)
		features.AppendElement(mechanisms)
	}
	dbBack := xmpp.NewElementNamespace("dialback", dialbackNamespace)
	dbBack.AppendElement(xmpp.NewElementName("errors"))
	features.AppendElement(dbBack)

	s.setState(inConnected)
	s.writeElement(features)
}

func (s *inStream) handleConnected(elem xmpp.XElement) {
	if !s.isSecured() {
		s.proceedStartTLS(elem)
		return
	}
	if !s.isAuthenticated() && elem.Name() == "auth" {
		s.startAuthentication(elem)
		return
	}
	switch elem.Name() {
	case "db:result":
		s.authorizeDialbackKey(elem)

	case "db:verify":
		s.verifyDialbackKey(elem)

	default:
		switch elem := elem.(type) {
		case xmpp.Stanza:
			s.processStanza(elem)
		}
	}
}

func (s *inStream) processStanza(stanza xmpp.Stanza) {
	switch stanza := stanza.(type) {
	case *xmpp.Presence:
		s.processPresence(stanza)
	case *xmpp.IQ:
		s.processIQ(stanza)
	case *xmpp.Message:
		s.processMessage(stanza)
	}
}

func (s *inStream) processPresence(presence *xmpp.Presence) {
	// process roster presence
	if presence.ToJID().IsBare() {
		if r := s.mods.Roster; r != nil {
			s.mods.Roster.ProcessPresence(presence)
		}
		return
	}
	s.router.Route(presence)
}

func (s *inStream) processIQ(iq *xmpp.IQ) {
	s.router.Route(iq)
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
	default:
		// silently ignore it...
		break
	}
}

func (s *inStream) proceedStartTLS(elem xmpp.XElement) {
	if elem.Namespace() != tlsNamespace {
		s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
		return

	} else if elem.Name() != "starttls" {
		s.disconnectWithStreamError(streamerror.ErrNotAuthorized)
		return
	}
	s.writeElement(xmpp.NewElementNamespace("proceed", tlsNamespace))

	s.cfg.transport.StartTLS(&tls.Config{
		ServerName:   s.localDomain,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		Certificates: s.router.Certificates(),
	}, false)
	atomic.StoreUint32(&s.secured, 1)

	log.Infof("secured stream... id: %s", s.id)
	s.restartSession()
}

func (s *inStream) startAuthentication(elem xmpp.XElement) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
		return
	}
	if elem.Attributes().Get("mechanism") != "EXTERNAL" {
		s.failAuthentication("invalid-mechanism", "")
		return
	}
	// validate initiating server certificate
	certs := s.cfg.transport.PeerCertificates()
	for _, cert := range certs {
		for _, dnsName := range cert.DNSNames {
			if dnsName == s.remoteDomain {
				s.finishAuthentication()
				return
			}
		}
	}
	s.failAuthentication("bad-protocol", "failed to get peer certificate")
}

func (s *inStream) finishAuthentication() {
	log.Infof("s2s in stream authenticated")
	atomic.StoreUint32(&s.authenticated, 1)

	success := xmpp.NewElementNamespace("success", saslNamespace)
	s.writeElement(success)
	s.restartSession()
}

func (s *inStream) failAuthentication(reason, text string) {
	log.Infof("failed s2s in stream authentication: %s (text: %s)", reason, text)
	failure := xmpp.NewElementNamespace("failure", saslNamespace)
	failure.AppendElement(xmpp.NewElementName(reason))
	if len(text) > 0 {
		textEl := xmpp.NewElementName("text")
		textEl.SetText(text)
		failure.AppendElement(textEl)
	}
	s.writeElement(failure)
}

func (s *inStream) authorizeDialbackKey(elem xmpp.XElement) {
	if !s.router.IsLocalHost(elem.To()) {
		s.writeStanzaErrorResponse(elem, xmpp.ErrItemNotFound)
		return
	}
	log.Infof("authorizing dialback key: %s...", elem.Text())

	outCfg, err := s.cfg.dialer.dial(elem.To(), elem.From())
	if err != nil {
		log.Error(err)
		s.writeStanzaErrorResponse(elem, xmpp.ErrRemoteServerNotFound)
		return
	}
	// create verify element
	dbVerify := xmpp.NewElementName("db:verify")
	dbVerify.SetID(s.sess.StreamID())
	dbVerify.SetFrom(elem.To())
	dbVerify.SetTo(elem.From())
	dbVerify.SetText(elem.Text())
	outCfg.dbVerify = dbVerify

	outStm := newOutStream(s.router)
	outStm.start(outCfg)

	// wait remote server verification
	select {
	case valid := <-outStm.verify():
		reply := xmpp.NewElementName("db:result")
		reply.SetFrom(elem.To())
		reply.SetTo(elem.From())
		if valid {
			reply.SetType("valid")
			atomic.StoreUint32(&s.authenticated, 1)

		} else {
			reply.SetType("invalid")
		}
		s.writeElement(reply)
		outStm.Disconnect(nil)

	case <-outStm.done():
		// remote server closed connection unexpectedly
		s.writeStanzaErrorResponse(elem, xmpp.ErrRemoteServerTimeout)
		break
	}
}

func (s *inStream) verifyDialbackKey(elem xmpp.XElement) {
	if !s.router.IsLocalHost(elem.To()) {
		s.writeStanzaErrorResponse(elem, xmpp.ErrItemNotFound)
		return
	}
	dbVerify := xmpp.NewElementName("db:verify")
	dbVerify.SetFrom(elem.To())
	dbVerify.SetTo(elem.From())
	dbVerify.SetID(elem.ID())

	expectedKey := s.cfg.keyGen.generate(elem.From(), elem.To(), elem.ID())
	if expectedKey == elem.Text() {
		log.Infof("dialback key successfully verified... (key: %s)", elem.Text())
		dbVerify.SetType("valid")
	} else {
		log.Infof("failed dialback key verification... (expected: %s, got: %s)", expectedKey, elem.Text())
		dbVerify.SetType("invalid")
	}
	s.writeElement(dbVerify)
}

func (s *inStream) writeStanzaErrorResponse(elem xmpp.XElement, stanzaErr *xmpp.StanzaError) {
	resp := xmpp.NewElementFromElement(elem)
	resp.SetType(xmpp.ErrorType)
	resp.SetFrom(elem.To())
	resp.SetTo(elem.From())
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
	if s.getState() != inDisconnected {
		go s.doRead()
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

func (s *inStream) disconnect(err error) {
	if s.getState() == inDisconnected {
		return
	}
	switch err {
	case nil:
		s.disconnectClosingSession(false)
	default:
		if stmErr, ok := err.(*streamerror.Error); ok {
			s.disconnectWithStreamError(stmErr)
		} else {
			log.Error(err)
			s.disconnectClosingSession(false)
		}
	}
}

func (s *inStream) disconnectWithStreamError(err *streamerror.Error) {
	if s.getState() == inConnecting {
		s.sess.Open()
	}
	s.writeElement(err.Element())
	s.disconnectClosingSession(true)
}

func (s *inStream) disconnectClosingSession(closeSession bool) {
	if closeSession {
		s.sess.Close()
	}
	if s.cfg.onInDisconnect != nil {
		s.cfg.onInDisconnect(s)
	}

	s.setState(inDisconnected)
	s.cfg.transport.Close()
}

func (s *inStream) restartSession() {
	j, _ := jid.New("", s.cfg.localDomain, "", true)
	s.sess = session.New(s.id, &session.Config{
		JID:           j,
		Transport:     s.cfg.transport,
		MaxStanzaSize: s.cfg.maxStanzaSize,
		RemoteDomain:  s.remoteDomain,
		IsServer:      true,
	}, s.router)
	s.setState(inConnecting)
}

func (s *inStream) isSecured() bool {
	return atomic.LoadUint32(&s.secured) == 1
}

func (s *inStream) isAuthenticated() bool {
	return atomic.LoadUint32(&s.authenticated) == 1
}

func (s *inStream) setState(state uint32) {
	atomic.StoreUint32(&s.state, state)
}

func (s *inStream) getState() uint32 {
	return atomic.LoadUint32(&s.state)
}

var inStreamCounter uint64

func nextInID() string {
	return fmt.Sprintf("s2s:in:%d", atomic.AddUint64(&inStreamCounter, 1))
}
