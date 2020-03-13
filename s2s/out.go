/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"fmt"
	"sync/atomic"

	streamerror "github.com/ortuman/jackal/errors"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router/host"
	"github.com/ortuman/jackal/session"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/util/runqueue"
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
	id            string
	cfg           *outConfig
	runQueue      *runqueue.RunQueue
	hosts         *host.Hosts
	dialer        Dialer
	state         uint32
	tr            transport.Transport
	sess          *session.Session
	secured       uint32
	authenticated uint32
	pendingSendQ  []xmpp.XElement
	dbVerify      xmpp.XElement
	verifyCh      chan bool
	discCh        chan *streamerror.Error
}

func newOutStream(cfg *outConfig, hosts *host.Hosts, dialer Dialer) *outStream {
	id := nextOutID()
	return &outStream{
		id:       id,
		cfg:      cfg,
		hosts:    hosts,
		dialer:   dialer,
		state:    outDisconnected,
		discCh:   make(chan *streamerror.Error),
		runQueue: runqueue.New(id),
	}
}

func (s *outStream) ID() string {
	return s.cfg.localDomain + ":" + s.cfg.remoteDomain
}

func (s *outStream) SendElement(ctx context.Context, elem xmpp.XElement) {
	s.runQueue.Run(func() {
		s.sendElement(ctx, elem)
	})
}

func (s *outStream) Disconnect(ctx context.Context, err error) {
	waitCh := make(chan struct{})
	s.runQueue.Stop(func() {
		defer close(waitCh)
		if s.getState() == outDisconnected {
			return
		}
		s.disconnect(ctx, err)
	})
	<-waitCh
}

func (s *outStream) sendElement(ctx context.Context, elem xmpp.XElement) {
	switch s.getState() {
	case outVerified:
		s.writeElement(ctx, elem)
	case outDisconnected:
		if err := s.start(ctx); err != nil {
			log.Error(err)
			return
		}
		fallthrough
	default:
		// send element after verification has been completed
		s.pendingSendQ = append(s.pendingSendQ, elem)
		return
	}
}

func (s *outStream) verify(ctx context.Context, streamID, from, to, key string) <-chan bool {
	verifyCh := make(chan bool, 1)
	s.runQueue.Run(func() {
		dbVerify := xmpp.NewElementName("db:verify")
		dbVerify.SetID(streamID)
		dbVerify.SetFrom(from)
		dbVerify.SetTo(to)
		dbVerify.SetText(key)

		s.dbVerify = dbVerify
		s.verifyCh = verifyCh

		if err := s.start(ctx); err != nil {
			log.Error(err)
			return
		}
	})
	return verifyCh
}

func (s *outStream) done() <-chan *streamerror.Error { return s.discCh }

// runs on its own goroutine
func (s *outStream) doRead() {
	elem, sErr := s.sess.Receive()

	ctx, _ := context.WithTimeout(context.Background(), s.cfg.timeout)
	if sErr == nil {
		s.runQueue.Run(func() { s.readElement(ctx, elem) })
	} else {
		s.runQueue.Run(func() {
			if s.getState() == outDisconnected {
				return // already disconnected...
			}
			log.Infof("s2s out stream disconnected... (domainpair: %s)", s.ID())

			s.handleSessionError(ctx, sErr)
		})
	}
}

func (s *outStream) dial(ctx context.Context) error {
	conn, err := s.dialer.Dial(ctx, s.cfg.remoteDomain)
	if err != nil {
		return err
	}
	s.tr = transport.NewSocketTransport(conn, s.cfg.keepAlive)
	return nil
}

func (s *outStream) start(ctx context.Context) error {
	if err := s.dial(ctx); err != nil {
		return err
	}
	s.restartSession()

	go s.doRead() // start reading transport...

	_ = s.sess.Open(ctx, nil)
	return nil
}

func (s *outStream) handleElement(ctx context.Context, elem xmpp.XElement) {
	switch s.getState() {
	case outConnecting:
		s.handleConnecting()
	case outConnected:
		s.handleConnected(ctx, elem)
	case outSecuring:
		s.handleSecuring(ctx, elem)
	case outAuthenticating:
		s.handleAuthenticating(ctx, elem)
	case outValidatingDialbackKey:
		s.handleValidatingDialbackKey(ctx, elem)
	case outAuthorizingDialbackKey:
		s.handleAuthorizingDialbackKey(ctx, elem)
	}
}

func (s *outStream) handleConnecting() {
	s.setState(outConnected)
}

func (s *outStream) handleConnected(ctx context.Context, elem xmpp.XElement) {
	if elem.Name() != "stream:features" {
		s.disconnectWithStreamError(ctx, streamerror.ErrUnsupportedStanzaType)
		return
	}
	if !s.isSecured() {
		if elem.Elements().ChildrenNamespace("starttls", tlsNamespace) == nil {
			// unsecured channels not supported
			s.disconnectWithStreamError(ctx, streamerror.ErrPolicyViolation)
			return
		}
		s.setState(outSecuring)
		s.writeElement(ctx, xmpp.NewElementNamespace("starttls", tlsNamespace))

	} else {
		// authorize dialback key
		if s.dbVerify != nil {
			s.setState(outAuthorizingDialbackKey)
			s.writeElement(ctx, s.dbVerify)
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
				s.writeElement(ctx, auth)

			} else if elem.Elements().ChildrenNamespace("dialback", dialbackNamespace) != nil {
				s.setState(outValidatingDialbackKey)
				db := xmpp.NewElementName("db:result")
				db.SetFrom(s.cfg.localDomain)
				db.SetTo(s.cfg.remoteDomain)
				db.SetText(s.cfg.keyGen.generate(s.cfg.remoteDomain, s.cfg.localDomain, s.sess.StreamID()))
				s.writeElement(ctx, db)

			} else {
				// no verification mechanism found... do not allow remote connection
				s.disconnectWithStreamError(ctx, streamerror.ErrRemoteConnectionFailed)
			}
		} else {
			s.finishVerification(ctx)
		}
	}
}

func (s *outStream) handleSecuring(ctx context.Context, elem xmpp.XElement) {
	if elem.Name() != "proceed" {
		s.disconnectWithStreamError(ctx, streamerror.ErrUnsupportedStanzaType)
		return
	} else if elem.Namespace() != tlsNamespace {
		s.disconnectWithStreamError(ctx, streamerror.ErrInvalidNamespace)
		return
	}
	s.tr.StartTLS(s.cfg.tls, true)

	atomic.StoreUint32(&s.secured, 1)
	s.restartSession()

	_ = s.sess.Open(ctx, nil)
}

func (s *outStream) handleAuthenticating(ctx context.Context, elem xmpp.XElement) {
	if elem.Namespace() != saslNamespace {
		s.disconnectWithStreamError(ctx, streamerror.ErrInvalidNamespace)
		return
	}
	switch elem.Name() {
	case "success":
		atomic.StoreUint32(&s.authenticated, 1)
		s.restartSession()
		_ = s.sess.Open(ctx, nil)

	case "failure":
		s.disconnectWithStreamError(ctx, streamerror.ErrRemoteConnectionFailed)

	default:
		s.disconnectWithStreamError(ctx, streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *outStream) handleValidatingDialbackKey(ctx context.Context, elem xmpp.XElement) {
	switch elem.Name() {
	case "db:result":
		if elem.From() != s.cfg.remoteDomain {
			s.disconnectWithStreamError(ctx, streamerror.ErrInvalidFrom)
			return
		}
		switch elem.Type() {
		case "valid":
			log.Infof("s2s out stream successfully validated... (domainpair: %s)", s.ID())
			s.finishVerification(ctx)

		default:
			log.Infof("failed s2s out stream validation... (domainpair: %s)", s.ID())
			s.disconnectWithStreamError(ctx, streamerror.ErrRemoteConnectionFailed)
		}
	}
}

func (s *outStream) handleAuthorizingDialbackKey(ctx context.Context, elem xmpp.XElement) {
	switch elem.Name() {
	case "db:verify":
		s.verifyCh <- elem.Type() == "valid"

	default:
		s.disconnectWithStreamError(ctx, streamerror.ErrUnsupportedStanzaType)
	}
}

func (s *outStream) finishVerification(ctx context.Context) {
	s.setState(outVerified)

	// send pending elements...
	for _, el := range s.pendingSendQ {
		s.writeElement(ctx, el)
	}
	s.pendingSendQ = nil
}

func (s *outStream) writeStanzaErrorResponse(ctx context.Context, elem xmpp.XElement, stanzaErr *xmpp.StanzaError) {
	resp := xmpp.NewElementFromElement(elem)
	resp.SetType(xmpp.ErrorType)
	resp.SetFrom(elem.To())
	resp.SetTo(elem.From())
	resp.AppendElement(stanzaErr.Element())
	s.writeElement(ctx, resp)
}

func (s *outStream) writeElement(ctx context.Context, elem xmpp.XElement) {
	if err := s.sess.Send(ctx, elem); err != nil {
		log.Error(err)
	}
}

func (s *outStream) readElement(ctx context.Context, elem xmpp.XElement) {
	if elem != nil {
		s.handleElement(ctx, elem)
	}
	if s.getState() != outDisconnected {
		go s.doRead()
	}
}

func (s *outStream) handleSessionError(ctx context.Context, sErr *session.Error) {
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

func (s *outStream) disconnect(ctx context.Context, err error) {
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

func (s *outStream) disconnectWithStreamError(ctx context.Context, err *streamerror.Error) {
	// notify disconnection
	select {
	case s.discCh <- err:
		break
	default:
		break
	}
	s.writeElement(ctx, err.Element())
	s.disconnectClosingSession(ctx, true)
}

func (s *outStream) disconnectClosingSession(ctx context.Context, closeSession bool) {
	if closeSession {
		_ = s.sess.Close(ctx)
	}
	atomic.StoreUint32(&s.secured, 0)
	atomic.StoreUint32(&s.authenticated, 0)

	s.setState(outDisconnected)
	_ = s.tr.Close()
}

func (s *outStream) restartSession() {
	j, _ := jid.New("", s.cfg.localDomain, "", true)
	s.sess = session.New(s.id, &session.Config{
		JID:           j,
		Transport:     s.tr,
		MaxStanzaSize: s.cfg.maxStanzaSize,
		RemoteDomain:  s.cfg.remoteDomain,
		IsServer:      true,
		IsInitiating:  true,
	}, s.hosts)
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
