// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package s2s

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/jackal-xmpp/runqueue"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/cluster/kv"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	xmppparser "github.com/ortuman/jackal/parser"
	"github.com/ortuman/jackal/router/stream"
	xmppsession "github.com/ortuman/jackal/session"
	"github.com/ortuman/jackal/shaper"
	"github.com/ortuman/jackal/transport"
)

var (
	errServerTimeout = errors.New("s2s: remote server timeout")
)

type outType int8

const (
	defaultType = outType(iota)
	dialbackType
)

func (t outType) String() string {
	switch t {
	case dialbackType:
		return "dialback"
	}
	return "default"
}

type outS2SState uint32

const (
	outConnecting outS2SState = iota
	outConnected
	outSecuring
	outAuthenticating
	outAuthenticated
	outVerifyingDialbackKey
	outAuthorizingDialbackKey
	outDisconnected
)

// DialbackParams contains S2S dialback verification parameters.
type DialbackParams struct {
	// StreamID represents verification stream identifier.
	StreamID string

	// From represents verification sender domain.
	From string

	// To represents verification target domain.
	To string

	// Key is the dialback generated key.
	Key string
}

type outS2S struct {
	typ      outType
	sender   string
	target   string
	opts     Options
	tr       transport.Transport
	kv       kv.KV
	session  session
	dbParams DialbackParams
	dialer   dialer
	hosts    *host.Hosts
	tlsCfg   *tls.Config
	onClose  func(s *outS2S)
	dbResCh  chan stream.DialbackResult
	shapers  shaper.Shapers
	sn       *sonar.Sonar
	rq       *runqueue.RunQueue

	state        uint32
	flags        flags
	pendingQueue []stravaganza.Element
}

func newOutS2S(
	sender string,
	target string,
	tlsCfg *tls.Config,
	hosts *host.Hosts,
	opts Options,
	kv kv.KV,
	shapers shaper.Shapers,
	sn *sonar.Sonar,
	onClose func(s *outS2S),
) *outS2S {
	stm := &outS2S{
		typ:     defaultType,
		sender:  sender,
		target:  target,
		hosts:   hosts,
		tlsCfg:  tlsCfg,
		opts:    opts,
		onClose: onClose,
		kv:      kv,
		shapers: shapers,
		sn:      sn,
		dialer:  newDialer(opts.DialTimeout, tlsCfg),
	}
	stm.rq = runqueue.New(stm.ID().String(), log.Errorf)
	return stm
}

func newDialbackS2S(
	sender string,
	target string,
	tlsCfg *tls.Config,
	hosts *host.Hosts,
	opts Options,
	dbParams DialbackParams,
	shapers shaper.Shapers,
) *outS2S {
	stm := &outS2S{
		typ:      dialbackType,
		sender:   sender,
		target:   target,
		hosts:    hosts,
		tlsCfg:   tlsCfg,
		opts:     opts,
		dbParams: dbParams,
		dialer:   newDialer(opts.DialTimeout, tlsCfg),
		dbResCh:  make(chan stream.DialbackResult, 1),
		shapers:  shapers,
	}
	stm.rq = runqueue.New(stm.ID().String(), log.Errorf)
	return stm
}

func (s *outS2S) ID() stream.S2SOutID {
	return stream.S2SOutID{Sender: s.sender, Target: s.target}
}

func (s *outS2S) Done() <-chan stream.DialbackResult {
	return s.dbResCh
}

func (s *outS2S) SendElement(elem stravaganza.Element) <-chan error {
	errCh := make(chan error, 1)
	s.rq.Run(func() {
		ctx, cancel := s.requestContext()
		defer cancel()
		errCh <- s.sendOrEnqueueElement(ctx, elem)
	})
	return errCh
}

func (s *outS2S) Disconnect(streamErr *streamerror.Error) <-chan error {
	errCh := make(chan error, 1)
	s.rq.Run(func() {
		ctx, cancel := s.requestContext()
		defer cancel()
		errCh <- s.disconnect(ctx, streamErr)
	})
	return errCh
}

func (s *outS2S) dial(ctx context.Context) error {
	conn, usesTLS, err := s.dialer.DialContext(ctx, s.target)
	if err != nil {
		switch err := err.(type) {
		case net.Error:
			if err.Timeout() {
				return errServerTimeout
			}
		}
		return err
	}
	log.Infow("Dialed S2S remote connection", "target", s.target, "direct_tls", usesTLS)

	s.tr = transport.NewSocketTransport(conn)

	// set default rate limiter
	rLim := s.shapers.DefaultS2S().RateLimiter()
	if err := s.tr.SetReadRateLimiter(rLim); err != nil {
		return err
	}
	s.session = xmppsession.New(
		xmppsession.S2SSession,
		s.ID().String(),
		s.tr,
		s.hosts,
		xmppsession.Options{
			MaxStanzaSize: s.opts.MaxStanzaSize,
			IsOut:         true,
		},
	)
	// set target domain JID
	jd, _ := jid.New("", s.target, "", true)
	s.session.SetFromJID(jd)

	if usesTLS {
		s.flags.setSecured() // already secured
	}
	return nil
}

func (s *outS2S) start() error {
	s.restartSession()

	ctx, cancel := s.requestContext()
	_ = s.session.OpenStream(ctx, nil)

	switch s.typ {
	case defaultType:
		log.Infow("Registered S2S out stream", "sender", s.sender, "target", s.target)
	case dialbackType:
		log.Infow("Registered S2S dialback stream", "sender", s.sender, "target", s.target)
	}
	// post registered S2S event
	err := s.postStreamEvent(ctx, event.S2SOutStreamRegistered, &event.S2SStreamEventInfo{
		ID: s.ID().String(),
	})
	cancel()

	if err != nil {
		return err
	}
	reportOutgoingConnectionRegistered(s.typ)

	s.readLoop()
	return nil
}

func (s *outS2S) readLoop() {
	tm := time.AfterFunc(s.opts.KeepAlive, s.connTimeout)
	elem, sErr := s.session.Receive()
	tm.Stop()

	for {
		if s.getState() == outDisconnected {
			return
		}
		if sErr == xmppparser.ErrNoElement {
			goto doRead // continue reading
		}
		s.handleSessionResult(elem, sErr)

	doRead:
		if s.getState() != outAuthenticated {
			tm = time.AfterFunc(s.opts.KeepAlive, s.connTimeout) // schedule read timeout
		}
		elem, sErr = s.session.Receive()
		if tm != nil {
			tm.Stop()
			tm = nil
		}
	}
}

func (s *outS2S) handleSessionResult(elem stravaganza.Element, sErr error) {
	doneCh := make(chan struct{})
	s.rq.Run(func() {
		defer close(doneCh)

		ctx, cancel := s.requestContext()
		defer cancel()

		var err error
		if sErr != nil {
			err = s.handleSessionError(ctx, sErr)
		}
		if elem != nil {
			err = s.handleElement(ctx, elem)
		}
		if err != nil {
			log.Errorf("Failed to process outgoing S2S session result: %v", err)
			_ = s.close(ctx)
			return
		}
	})
	<-doneCh
}

func (s *outS2S) connTimeout() {
	s.rq.Run(func() {
		ctx, cancel := s.requestContext()
		defer cancel()
		_ = s.disconnect(ctx, streamerror.E(streamerror.ConnectionTimeout))
	})
}

func (s *outS2S) handleElement(ctx context.Context, elem stravaganza.Element) error {
	var err error
	t0 := time.Now()
	switch s.getState() {
	case outConnecting:
		err = s.handleConnecting(ctx, elem)
	case outConnected:
		err = s.handleConnected(ctx, elem)
	case outSecuring:
		err = s.handleSecuring(ctx, elem)
	case outAuthenticating:
		err = s.handleAuthenticating(ctx, elem)
	case outVerifyingDialbackKey:
		err = s.handleVerifyingDialbackKey(ctx, elem)
	case outAuthorizingDialbackKey:
		err = s.handleAuthorizingDialbackKey(ctx, elem)
	}
	reportIncomingRequest(
		elem.Name(),
		elem.Attribute(stravaganza.Type),
		time.Since(t0).Seconds(),
	)
	return err
}

func (s *outS2S) handleConnecting(_ context.Context, _ stravaganza.Element) error {
	s.setState(outConnected)
	return nil
}

func (s *outS2S) handleConnected(ctx context.Context, elem stravaganza.Element) error {
	if elem.Name() != "stream:features" {
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	}
	if !s.flags.isSecured() {
		if elem.ChildNamespace("starttls", tlsNamespace) == nil {
			// unsecured connections are unsupported
			return s.disconnect(ctx, streamerror.E(streamerror.PolicyViolation))
		}
		s.setState(outSecuring)

		startTLS := stravaganza.NewBuilder("starttls").
			WithAttribute(stravaganza.Namespace, tlsNamespace).
			Build()
		return s.sendElement(ctx, startTLS)
	}
	if s.flags.isAuthenticated() {
		return s.finishAuthentication(ctx)
	}
	switch s.typ {
	case defaultType:
		switch {
		case hasExternalAuthMechanism(elem):
			s.setState(outAuthenticating)
			return s.sendElement(ctx, stravaganza.NewBuilder("auth").
				WithAttribute(stravaganza.Namespace, saslNamespace).
				WithAttribute("mechanism", "EXTERNAL").
				WithText("=").
				Build(),
			)

		case hasDialbackFeature(elem):
			streamID := s.session.StreamID()

			// register dialback request
			if err := registerDbRequest(ctx, s.target, s.sender, streamID, s.kv); err != nil {
				return err
			}
			s.setState(outVerifyingDialbackKey)
			return s.sendElement(ctx, stravaganza.NewBuilder("db:result").
				WithAttribute(stravaganza.From, s.sender).
				WithAttribute(stravaganza.To, s.target).
				WithText(
					dbKey(
						s.opts.DialbackSecret,
						s.target,
						s.sender,
						streamID,
					),
				).
				Build(),
			)

		default:
			return s.disconnect(ctx, streamerror.E(streamerror.RemoteConnectionFailed))
		}

	case dialbackType:
		s.setState(outAuthorizingDialbackKey)
		return s.sendElement(ctx, stravaganza.NewBuilder("db:verify").
			WithAttribute(stravaganza.ID, s.dbParams.StreamID).
			WithAttribute(stravaganza.From, s.dbParams.From).
			WithAttribute(stravaganza.To, s.dbParams.To).
			WithText(s.dbParams.Key).
			Build(),
		)
	}
	return nil
}

func (s *outS2S) handleSecuring(ctx context.Context, elem stravaganza.Element) error {
	if elem.Name() != "proceed" {
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	} else if elem.Attribute(stravaganza.Namespace) != tlsNamespace {
		return s.disconnect(ctx, streamerror.E(streamerror.InvalidNamespace))
	}
	// proceed with TLS securing
	s.tr.StartTLS(s.tlsCfg, true)

	s.flags.setSecured()
	s.restartSession()

	return s.session.OpenStream(ctx, nil)
}

func (s *outS2S) handleAuthenticating(ctx context.Context, elem stravaganza.Element) error {
	if elem.Attribute(stravaganza.Namespace) != saslNamespace {
		return s.disconnect(ctx, streamerror.E(streamerror.InvalidNamespace))
	}
	switch elem.Name() {
	case "success":
		s.flags.setAuthenticated()

		s.restartSession()
		return s.session.OpenStream(ctx, nil)

	case "failure":
		return s.disconnect(ctx, streamerror.E(streamerror.RemoteConnectionFailed))

	default:
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	}
}

func (s *outS2S) handleVerifyingDialbackKey(ctx context.Context, elem stravaganza.Element) error {
	switch elem.Name() {
	case "db:result":
		switch elem.Attribute(stravaganza.Type) {
		case "valid":
			log.Infow("S2S dialback key successfully verified", "from", s.sender, "to", s.target)
			return s.finishAuthentication(ctx)

		default:
			log.Infow("Failed to verify S2S dialback key", "from", s.sender, "to", s.target)
			return s.disconnect(ctx, streamerror.E(streamerror.RemoteConnectionFailed))
		}

	default:
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	}
}

func (s *outS2S) handleAuthorizingDialbackKey(ctx context.Context, elem stravaganza.Element) error {
	switch elem.Name() {
	case "db:verify":
		typ := elem.Attribute(stravaganza.Type)
		isValid := typ == "valid"

		s.dbResCh <- stream.DialbackResult{
			Valid: isValid,
			Error: elem.Child("error"),
		}
		return s.disconnect(ctx, nil)

	default:
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	}
}

func (s *outS2S) handleSessionError(ctx context.Context, err error) error {
	switch err {
	case nil, io.EOF, io.ErrUnexpectedEOF:
		return s.close(ctx)

	case xmppparser.ErrStreamClosedByPeer:
		_ = s.session.Close(ctx)
		return s.close(ctx)

	default:
		switch err := err.(type) {
		case *streamerror.Error:
			return s.disconnect(ctx, err)

		case *stanzaerror.Error:
			return s.sendElement(ctx, err.Element())

		default:
			return err
		}
	}
}

func (s *outS2S) finishAuthentication(ctx context.Context) error {
	s.setState(outAuthenticated)

	// send pending elements
	for _, elem := range s.pendingQueue {
		if err := s.sendElement(ctx, elem); err != nil {
			return err
		}
	}
	s.pendingQueue = nil
	return nil
}

func (s *outS2S) restartSession() {
	_ = s.session.Reset(s.tr)
	s.setState(outConnecting)
}

func (s *outS2S) disconnect(ctx context.Context, streamErr *streamerror.Error) error {
	if s.getState() == outConnecting {
		_ = s.session.OpenStream(ctx, nil)
	}
	if streamErr != nil {
		if err := s.sendElement(ctx, streamErr.Element()); err != nil {
			return err
		}
	}
	_ = s.session.Close(ctx)
	return s.close(ctx)
}

func (s *outS2S) sendOrEnqueueElement(ctx context.Context, elem stravaganza.Element) error {
	switch s.getState() {
	case outAuthenticated:
		return s.sendElement(ctx, elem)
	default:
		s.pendingQueue = append(s.pendingQueue, elem)
	}
	return nil
}

func (s *outS2S) sendElement(ctx context.Context, elem stravaganza.Element) error {
	err := s.session.Send(ctx, elem)

	switch stanza := elem.(type) {
	case stravaganza.Stanza:
		// post S2S stanza sent event
		err := s.postStreamEvent(ctx, event.S2SOutStreamStanzaSent, &event.S2SStreamEventInfo{
			ID:     s.ID().String(),
			Sender: s.sender,
			Target: s.target,
			Stanza: stanza,
		})
		if err != nil {
			return err
		}
	}
	reportOutgoingRequest(
		elem.Name(),
		elem.Attribute(stravaganza.Type),
	)
	return err
}

func (s *outS2S) close(ctx context.Context) error {
	// unregister S2S out stream
	s.setState(outDisconnected)

	if s.onClose != nil {
		s.onClose(s)
	}
	if s.dbResCh != nil {
		close(s.dbResCh)
	}
	if s.typ == defaultType {
		log.Infow("Unregistered S2S out stream", "sender", s.sender, "target", s.target)
	}
	// post unregistered S2S event
	err := s.postStreamEvent(ctx, event.S2SOutStreamUnregistered, &event.S2SStreamEventInfo{
		ID: s.ID().String(),
	})
	if err != nil {
		return err
	}
	reportOutgoingConnectionUnregistered(s.typ)

	// close underlying transport
	return s.tr.Close()
}

func (s *outS2S) setState(state outS2SState) {
	atomic.StoreUint32(&s.state, uint32(state))
}

func (s *outS2S) getState() outS2SState {
	return outS2SState(atomic.LoadUint32(&s.state))
}

func (s *outS2S) postStreamEvent(ctx context.Context, eventName string, inf *event.S2SStreamEventInfo) error {
	if s.typ == dialbackType {
		return nil
	}
	return s.sn.Post(ctx, sonar.NewEventBuilder(eventName).
		WithInfo(inf).
		WithSender(s).
		Build(),
	)
}

func (s *outS2S) requestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.opts.RequestTimeout)
}

func hasExternalAuthMechanism(streamFeatures stravaganza.Element) bool {
	mechanisms := streamFeatures.ChildNamespace("mechanisms", saslNamespace)
	if mechanisms == nil {
		return false
	}
	for _, m := range mechanisms.AllChildren() {
		if m.Name() == "mechanism" && m.Text() == "EXTERNAL" {
			return true
		}
	}
	return false
}

func hasDialbackFeature(streamFeatures stravaganza.Element) bool {
	return streamFeatures.ChildrenNamespace("dialback", dialbackNamespace) != nil
}
