/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync/atomic"
	"time"

	streamerror "github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/runqueue"
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
	runQueue      *runqueue.RunQueue
}

func newInStream(config *streamConfig, mods *module.Modules, router *router.Router) *inStream {
	id := nextInID()
	s := &inStream{
		id:       id,
		cfg:      config,
		router:   router,
		mods:     mods,
		runQueue: runqueue.New(id),
	}
	// start s2s in session
	s.restartSession()

	if config.connectTimeout > 0 {
		s.connectTm = time.AfterFunc(config.connectTimeout, s.connectTimeout)
	}
	go s.doRead() // start reading transport...
	return s
}

func (s *inStream) ID() string {
	return s.id
}

func (s *inStream) Disconnect(ctx context.Context, err error) {
	if s.getState() == inDisconnected {
		return
	}
	waitCh := make(chan struct{})
	s.runQueue.Run(func() {
		s.disconnect(ctx, err)
		close(waitCh)
	})
	<-waitCh
}

func (s *inStream) connectTimeout() {
	s.runQueue.Run(func() {
		ctx, _ := context.WithTimeout(context.Background(), s.cfg.timeout)
		s.disconnect(ctx, streamerror.ErrConnectionTimeout)
	})
}

// runs on its own goroutine
func (s *inStream) doRead() {
	elem, sErr := s.sess.Receive()

	ctx, _ := context.WithTimeout(context.Background(), s.cfg.timeout)
	if sErr == nil {
		s.runQueue.Run(func() {
			s.readElement(ctx, elem)
		})
	} else {
		s.runQueue.Run(func() {
			if s.getState() == inDisconnected {
				return // already disconnected...
			}
			s.handleSessionError(ctx, sErr)
		})
	}
}

func (s *inStream) handleElement(ctx context.Context, elem xmpp.XElement) {
	switch s.getState() {
	case inConnecting:
		s.handleConnecting(ctx, elem)
	case inConnected:
		s.handleConnected(ctx, elem)
	}
}

func (s *inStream) handleConnecting(ctx context.Context, elem xmpp.XElement) {
	// cancel connection timeout timer
	if s.connectTm != nil {
		s.connectTm.Stop()
		s.connectTm = nil
	}
	// assign domain pair
	s.localDomain = s.router.DefaultHostName()
	s.remoteDomain = elem.From()

	// open stream session
	s.sess.SetRemoteDomain(s.remoteDomain)

	j, _ := jid.New("", s.localDomain, "", true)
	s.sess.SetJID(j)

	features := xmpp.NewElementName("stream:features")
	features.SetAttribute("xmlns:stream", streamNamespace)
	features.SetAttribute("version", "1.0")

	if !s.isSecured() {
		starttls := xmpp.NewElementNamespace("starttls", tlsNamespace)
		starttls.AppendElement(xmpp.NewElementName("required"))
		features.AppendElement(starttls)
		s.setState(inConnected)
		_ = s.sess.Open(ctx, features)
		return
	}

	_ = s.sess.Open(ctx, nil)

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
	s.writeElement(ctx, features)
}

func (s *inStream) handleConnected(ctx context.Context, elem xmpp.XElement) {
	if !s.isSecured() {
		s.proceedStartTLS(ctx, elem)
		return
	}
	if !s.isAuthenticated() && elem.Name() == "auth" {
		s.startAuthentication(ctx, elem)
		return
	}
	switch elem.Name() {
	case "db:result":
		s.authorizeDialbackKey(ctx, elem)

	case "db:verify":
		s.verifyDialbackKey(ctx, elem)

	default:
		switch elem := elem.(type) {
		case xmpp.Stanza:
			s.processStanza(ctx, elem)
		}
	}
}

func (s *inStream) processStanza(ctx context.Context, stanza xmpp.Stanza) {
	switch stanza := stanza.(type) {
	case *xmpp.Presence:
		s.processPresence(ctx, stanza)
	case *xmpp.IQ:
		s.processIQ(ctx, stanza)
	case *xmpp.Message:
		s.processMessage(ctx, stanza)
	}
}

func (s *inStream) processPresence(ctx context.Context, presence *xmpp.Presence) {
	// process roster presence
	if presence.ToJID().IsBare() {
		if r := s.mods.Roster; r != nil {
			r.ProcessPresence(ctx, presence)
			return
		}
	}
	_ = s.router.Route(ctx, presence)
}

func (s *inStream) processIQ(ctx context.Context, iq *xmpp.IQ) {
	toJID := iq.ToJID()

	replyOnBehalf := !toJID.IsFullWithUser() && s.router.IsLocalHost(toJID.Domain())
	if !replyOnBehalf {
		switch s.router.Route(ctx, iq) {
		case router.ErrResourceNotFound:
			s.writeElement(ctx, iq.ServiceUnavailableError())
		case router.ErrFailedRemoteConnect:
			s.writeElement(ctx, iq.RemoteServerNotFoundError())
		case router.ErrBlockedJID:
			// Destination user is a blocked JID
			if iq.IsGet() || iq.IsSet() {
				s.writeElement(ctx, iq.ServiceUnavailableError())
			}
		}
		return
	}
	s.mods.ProcessIQ(ctx, iq)
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
	default:
		// silently ignore it...
		break
	}
}

func (s *inStream) proceedStartTLS(ctx context.Context, elem xmpp.XElement) {
	if elem.Namespace() != tlsNamespace {
		s.disconnectWithStreamError(ctx, streamerror.ErrInvalidNamespace)
		return

	} else if elem.Name() != "starttls" {
		s.disconnectWithStreamError(ctx, streamerror.ErrNotAuthorized)
		return
	}
	s.writeElement(ctx, xmpp.NewElementNamespace("proceed", tlsNamespace))

	s.cfg.transport.StartTLS(&tls.Config{
		ServerName:   s.localDomain,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		Certificates: s.router.Certificates(),
	}, false)
	atomic.StoreUint32(&s.secured, 1)

	log.Infof("secured stream... id: %s", s.id)
	s.restartSession()
}

func (s *inStream) startAuthentication(ctx context.Context, elem xmpp.XElement) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(ctx, streamerror.ErrInvalidNamespace)
		return
	}
	if elem.Attributes().Get("mechanism") != "EXTERNAL" {
		s.failAuthentication(ctx, "invalid-mechanism", "")
		return
	}
	// validate initiating server certificate
	certs := s.cfg.transport.PeerCertificates()
	for _, cert := range certs {
		for _, dnsName := range cert.DNSNames {
			if dnsName == s.remoteDomain {
				s.finishAuthentication(ctx)
				return
			}
		}
	}
	s.failAuthentication(ctx, "bad-protocol", "failed to get peer certificate")
}

func (s *inStream) finishAuthentication(ctx context.Context) {
	log.Infof("s2s in stream authenticated")
	atomic.StoreUint32(&s.authenticated, 1)

	success := xmpp.NewElementNamespace("success", saslNamespace)
	s.writeElement(ctx, success)
	s.restartSession()
}

func (s *inStream) failAuthentication(ctx context.Context, reason, text string) {
	log.Infof("failed s2s in stream authentication: %s (text: %s)", reason, text)
	failure := xmpp.NewElementNamespace("failure", saslNamespace)
	failure.AppendElement(xmpp.NewElementName(reason))
	if len(text) > 0 {
		textEl := xmpp.NewElementName("text")
		textEl.SetText(text)
		failure.AppendElement(textEl)
	}
	s.writeElement(ctx, failure)
}

func (s *inStream) authorizeDialbackKey(ctx context.Context, elem xmpp.XElement) {
	if !s.router.IsLocalHost(elem.To()) {
		s.writeStanzaErrorResponse(ctx, elem, xmpp.ErrItemNotFound)
		return
	}
	log.Infof("authorizing dialback key: %s...", elem.Text())

	outCfg, err := s.cfg.dialer.dial(s.router.DefaultHostName(), elem.From())
	if err != nil {
		log.Error(err)
		s.writeStanzaErrorResponse(ctx, elem, xmpp.ErrRemoteServerNotFound)
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
	_ = outStm.start(ctx, outCfg)

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
		s.writeElement(ctx, reply)
		outStm.Disconnect(ctx, nil)

	case <-outStm.done():
		// remote server closed connection unexpectedly
		s.writeStanzaErrorResponse(ctx, elem, xmpp.ErrRemoteServerTimeout)
		break
	}
}

func (s *inStream) verifyDialbackKey(ctx context.Context, elem xmpp.XElement) {
	if !s.router.IsLocalHost(elem.To()) {
		s.writeStanzaErrorResponse(ctx, elem, xmpp.ErrItemNotFound)
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
	s.writeElement(ctx, dbVerify)
}

func (s *inStream) writeStanzaErrorResponse(ctx context.Context, elem xmpp.XElement, stanzaErr *xmpp.StanzaError) {
	resp := xmpp.NewElementFromElement(elem)
	resp.SetType(xmpp.ErrorType)
	resp.SetFrom(elem.To())
	resp.SetTo(elem.From())
	resp.AppendElement(stanzaErr.Element())
	s.writeElement(ctx, resp)
}

func (s *inStream) writeElement(ctx context.Context, elem xmpp.XElement) {
	s.sess.Send(ctx, elem)
}

func (s *inStream) readElement(ctx context.Context, elem xmpp.XElement) {
	if elem != nil {
		s.handleElement(ctx, elem)
	}
	if s.getState() != inDisconnected {
		go s.doRead()
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

func (s *inStream) disconnect(ctx context.Context, err error) {
	if s.getState() == inDisconnected {
		return
	}
	switch err {
	case nil:
		s.disconnectClosingSession(ctx, false)
	default:
		if stmErr, ok := err.(*streamerror.Error); ok {
			s.disconnectWithStreamError(ctx, stmErr)
		} else {
			log.Error(err)
			s.disconnectClosingSession(ctx, false)
		}
	}
}

func (s *inStream) disconnectWithStreamError(ctx context.Context, err *streamerror.Error) {
	if s.getState() == inConnecting {
		_ = s.sess.Open(ctx, nil)
	}
	s.writeElement(ctx, err.Element())
	s.disconnectClosingSession(ctx, true)
}

func (s *inStream) disconnectClosingSession(ctx context.Context, closeSession bool) {
	if closeSession {
		_ = s.sess.Close(ctx)
	}
	if s.cfg.onInDisconnect != nil {
		s.cfg.onInDisconnect(s)
	}

	s.setState(inDisconnected)
	_ = s.cfg.transport.Close()

	s.runQueue.Stop(nil) // stop processing messages
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
