/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"fmt"
	"sync/atomic"

	"github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/session"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	outConnecting uint32 = iota
	outConnected
	outSecuring
	outAuthenticating
	outValidatingDialbackKey
	outAuthorizingDialbackKey
	outVerified
	outDisconnected
)

type outStream struct {
	started       uint32
	id            string
	cfg           *streamConfig
	router        *router.Router
	state         uint32
	sess          *session.Session
	secured       uint32
	authenticated uint32
	actorCh       chan func()
	sendQueue     []xmpp.XElement
	verified      chan xmpp.XElement
	verifyCh      chan bool
	discCh        chan *streamerror.Error
	onDisconnect  func(s stream.S2SOut)
}

func newOutStream(router *router.Router) *outStream {
	return &outStream{
		id:       nextOutID(),
		router:   router,
		actorCh:  make(chan func(), streamMailboxSize),
		verifyCh: make(chan bool, 1),
		discCh:   make(chan *streamerror.Error, 1),
	}
}

func (s *outStream) ID() string {
	return s.cfg.localDomain + ":" + s.cfg.remoteDomain
}

func (s *outStream) SendElement(elem xmpp.XElement) {
	if s.getState() == outDisconnected {
		return
	}
	s.actorCh <- func() {
		if s.getState() != outVerified {
			// send element after verification has been completed
			s.sendQueue = append(s.sendQueue, elem)
			return
		}
		s.writeElement(elem)
	}
}

func (s *outStream) Disconnect(err error) {
	if s.getState() == outDisconnected {
		return
	}
	waitCh := make(chan struct{})
	s.actorCh <- func() {
		s.disconnect(err)
		close(waitCh)
	}
	<-waitCh
}

func (s *outStream) start(cfg *streamConfig) error {
	if cfg.dbVerify != nil && cfg.dbVerify.Name() != "db:verify" {
		return fmt.Errorf("wrong dialback verification element name: %s", cfg.dbVerify.Name())
	}
	if !atomic.CompareAndSwapUint32(&s.started, 0, 1) {
		return fmt.Errorf("stream already started (domainpair: %s)", s.ID())
	}
	s.cfg = cfg

	// start s2s out session
	s.restartSession()

	go s.loop()
	go s.doRead() // start reading transport...

	s.actorCh <- func() {
		s.sess.Open()
	}
	return nil
}

func (s *outStream) verify() <-chan bool {
	return s.verifyCh
}

func (s *outStream) done() <-chan *streamerror.Error {
	return s.discCh
}

// runs on its own goroutine
func (s *outStream) loop() {
	for {
		f := <-s.actorCh
		f()
		if s.getState() == outDisconnected {
			return
		}
	}
}

// runs on its own goroutine
func (s *outStream) doRead() {
	if elem, sErr := s.sess.Receive(); sErr == nil {
		s.actorCh <- func() {
			s.readElement(elem)
		}
	} else {
		s.actorCh <- func() {
			if s.getState() == outDisconnected {
				return // already disconnected...
			}
			s.handleSessionError(sErr)
		}
	}
}

func (s *outStream) handleElement(elem xmpp.XElement) {
	switch s.getState() {
	case outConnecting:
		s.handleConnecting(elem)
	case outConnected:
		s.handleConnected(elem)
	case outSecuring:
		s.handleSecuring(elem)
	case outAuthenticating:
		s.handleAuthenticating(elem)
	case outValidatingDialbackKey:
		s.handleValidatingDialbackKey(elem)
	case outAuthorizingDialbackKey:
		s.handleAuthorizingDialbackKey(elem)
	}
}

func (s *outStream) handleConnecting(elem xmpp.XElement) {
	s.setState(outConnected)
}

func (s *outStream) handleConnected(elem xmpp.XElement) {
	if elem.Name() != "stream:features" {
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
		return
	}
	if !s.isSecured() {
		if elem.Elements().ChildrenNamespace("starttls", tlsNamespace) == nil {
			// unsecured channels not supported
			s.disconnectWithStreamError(streamerror.ErrPolicyViolation)
			return
		}
		s.setState(outSecuring)
		s.writeElement(xmpp.NewElementNamespace("starttls", tlsNamespace))

	} else {
		// authorize dialback key
		if s.cfg.dbVerify != nil {
			s.setState(outAuthorizingDialbackKey)
			s.writeElement(s.cfg.dbVerify)
			return
		}
		if !s.isAuthenticated() {
			var hasExternalAuth bool
			if mechanisms := elem.Elements().ChildNamespace("mechanisms", saslNamespace); mechanisms != nil {
				for _, m := range mechanisms.Elements().All() {
					if m.Name() == "mechanism" && m.Text() == "EXTERNAL" {
						hasExternalAuth = true
						break
					}
				}
			}
			if hasExternalAuth {
				s.setState(outAuthenticating)
				auth := xmpp.NewElementNamespace("auth", saslNamespace)
				auth.SetAttribute("mechanism", "EXTERNAL")
				auth.SetText("=")
				s.writeElement(auth)

			} else if elem.Elements().ChildrenNamespace("dialback", dialbackNamespace) != nil {
				s.setState(outValidatingDialbackKey)
				db := xmpp.NewElementName("db:result")
				db.SetFrom(s.cfg.localDomain)
				db.SetTo(s.cfg.remoteDomain)
				db.SetText(s.cfg.keyGen.generate(s.cfg.remoteDomain, s.cfg.localDomain, s.sess.StreamID()))
				s.writeElement(db)

			} else {
				// no verification mechanism found... do not allow remote connection
				s.disconnectWithStreamError(streamerror.ErrRemoteConnectionFailed)
			}
		} else {
			s.finishVerification()
		}
	}
}

func (s *outStream) handleSecuring(elem xmpp.XElement) {
	if elem.Name() != "proceed" {
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
		return
	} else if elem.Namespace() != tlsNamespace {
		s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
		return
	}
	s.cfg.transport.StartTLS(s.cfg.tls, true)

	atomic.StoreUint32(&s.secured, 1)
	s.restartSession()
	s.sess.Open()
}

func (s *outStream) handleAuthenticating(elem xmpp.XElement) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(streamerror.ErrInvalidNamespace)
		return
	}
	switch elem.Name() {
	case "success":
		atomic.StoreUint32(&s.authenticated, 1)
		s.restartSession()
		s.sess.Open()

	case "failure":
		s.disconnectWithStreamError(streamerror.ErrRemoteConnectionFailed)

	default:
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *outStream) handleValidatingDialbackKey(elem xmpp.XElement) {
	switch elem.Name() {
	case "db:result":
		if elem.From() != s.cfg.remoteDomain {
			s.disconnectWithStreamError(streamerror.ErrInvalidFrom)
			return
		}
		switch elem.Type() {
		case "valid":
			log.Infof("s2s out stream successfully validated... (domainpair: %s)", s.ID())
			s.finishVerification()

		default:
			log.Infof("failed s2s out stream validation... (domainpair: %s)", s.ID())
			s.disconnectWithStreamError(streamerror.ErrRemoteConnectionFailed)
		}
	}
}

func (s *outStream) handleAuthorizingDialbackKey(elem xmpp.XElement) {
	switch elem.Name() {
	case "db:verify":
		s.verifyCh <- elem.Type() == "valid"

	default:
		s.disconnectWithStreamError(streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *outStream) finishVerification() {
	// send pending elements...
	for _, el := range s.sendQueue {
		s.writeElement(el)
	}
	s.sendQueue = nil
	s.setState(outVerified)
}

func (s *outStream) writeStanzaErrorResponse(elem xmpp.XElement, stanzaErr *xmpp.StanzaError) {
	resp := xmpp.NewElementFromElement(elem)
	resp.SetType(xmpp.ErrorType)
	resp.SetFrom(elem.To())
	resp.SetTo(elem.From())
	resp.AppendElement(stanzaErr.Element())
	s.writeElement(resp)
}

func (s *outStream) writeElement(elem xmpp.XElement) {
	s.sess.Send(elem)
}

func (s *outStream) readElement(elem xmpp.XElement) {
	if elem != nil {
		s.handleElement(elem)
	}
	if s.getState() != outDisconnected {
		go s.doRead()
	}
}

func (s *outStream) handleSessionError(sErr *session.Error) {
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

func (s *outStream) disconnect(err error) {
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

func (s *outStream) disconnectWithStreamError(err *streamerror.Error) {
	s.discCh <- err
	s.writeElement(err.Element())
	s.disconnectClosingSession(true)
}

func (s *outStream) disconnectClosingSession(closeSession bool) {
	if closeSession {
		s.sess.Close()
	}
	if s.cfg.onOutDisconnect != nil {
		s.cfg.onOutDisconnect(s)
	}

	s.setState(outDisconnected)
	s.cfg.transport.Close()

	close(s.discCh)
}

func (s *outStream) restartSession() {
	j, _ := jid.New("", s.cfg.localDomain, "", true)
	s.sess = session.New(s.id, &session.Config{
		JID:           j,
		Transport:     s.cfg.transport,
		MaxStanzaSize: s.cfg.maxStanzaSize,
		RemoteDomain:  s.cfg.remoteDomain,
		IsServer:      true,
		IsInitiating:  true,
	}, s.router)
	s.setState(outConnecting)
}

func (s *outStream) isSecured() bool {
	return atomic.LoadUint32(&s.secured) == 1
}

func (s *outStream) isAuthenticated() bool {
	return atomic.LoadUint32(&s.authenticated) == 1
}

func (s *outStream) setState(state uint32) {
	atomic.StoreUint32(&s.state, state)
}

func (s *outStream) getState() uint32 {
	return atomic.LoadUint32(&s.state)
}

var outStreamCounter uint64

func nextOutID() string {
	return fmt.Sprintf("s2s:out:%d", atomic.AddUint64(&outStreamCounter, 1))
}
