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
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackal-xmpp/runqueue/v2"
	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/ortuman/jackal/pkg/component"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/module"
	xmppparser "github.com/ortuman/jackal/pkg/parser"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
	xmppsession "github.com/ortuman/jackal/pkg/session"
	"github.com/ortuman/jackal/pkg/shaper"
	"github.com/ortuman/jackal/pkg/transport"
)

type inState uint32

const (
	inConnecting inState = iota
	inConnected
	inAuthorizingDialbackKey
	inDisconnected
)

var inDisconnectTimeout = time.Second * 5

type inConfig struct {
	connectTimeout   time.Duration
	keepAliveTimeout time.Duration
	reqTimeout       time.Duration
	maxStanzaSize    int
	directTLS        bool
	tlsConfig        *tls.Config
}

type inS2S struct {
	id           stream.S2SInID
	cfg          inConfig
	tr           transport.Transport
	session      session
	hosts        hosts
	router       router.Router
	comps        components
	mods         modules
	outProvider  outProvider
	inHub        *InHub
	kv           kv.KV
	shapers      shaper.Shapers
	hk           *hook.Hooks
	rq           *runqueue.RunQueue
	discTm       *time.Timer
	doneCh       chan struct{}
	sendDisabled bool

	mu     sync.RWMutex
	state  inState
	flags  flags
	jd     *jid.JID
	target string
	sender string
}

func newInS2S(
	tr transport.Transport,
	hosts *host.Hosts,
	router router.Router,
	comps *component.Components,
	mods *module.Modules,
	outProvider *OutProvider,
	inHub *InHub,
	kv kv.KV,
	shapers shaper.Shapers,
	hk *hook.Hooks,
	cfg inConfig,
) (*inS2S, error) {
	// set default rate limiter
	rLim := shapers.DefaultS2S().RateLimiter()
	if err := tr.SetReadRateLimiter(rLim); err != nil {
		return nil, err
	}
	// create session
	id := nextStreamID()
	session := xmppsession.New(
		xmppsession.S2SSession,
		id.String(),
		tr,
		hosts,
		xmppsession.Config{
			MaxStanzaSize: cfg.maxStanzaSize,
		},
	)
	// init stream
	stm := &inS2S{
		id:          id,
		cfg:         cfg,
		tr:          tr,
		session:     session,
		hosts:       hosts,
		router:      router,
		comps:       comps,
		mods:        mods,
		outProvider: outProvider,
		inHub:       inHub,
		kv:          kv,
		shapers:     shapers,
		hk:          hk,
		rq:          runqueue.New(id.String()),
		doneCh:      make(chan struct{}),
		state:       inConnecting,
	}
	if cfg.directTLS {
		stm.flags.setSecured() // stream already secured
	}
	return stm, nil
}

func (s *inS2S) ID() stream.S2SInID {
	return s.id
}

func (s *inS2S) Disconnect(streamErr *streamerror.Error) <-chan error {
	errCh := make(chan error, 1)
	s.rq.Run(func() {
		ctx, cancel := s.requestContext()
		defer cancel()
		errCh <- s.disconnect(ctx, streamErr)
	})
	return errCh
}

func (s *inS2S) Done() <-chan struct{} {
	return s.doneCh
}

func (s *inS2S) start() error {
	s.inHub.register(s)

	log.Infow("registered S2S incoming stream", "id", s.id)

	// post registered incoming S2S event
	ctx, cancel := s.requestContext()
	_, err := s.runHook(ctx, hook.S2SInStreamRegistered, &hook.S2SStreamInfo{
		ID: s.ID().String(),
	})
	cancel()

	if err != nil {
		return err
	}
	reportIncomingConnectionRegistered()

	s.readLoop()
	return nil
}

func (s *inS2S) readLoop() {
	s.restartSession()

	tm := time.AfterFunc(s.cfg.connectTimeout, s.connTimeout) // schedule connect timeout
	elem, sErr := s.session.Receive()
	tm.Stop()

	for {
		if s.getState() == inDisconnected {
			return
		}
		if sErr == xmppparser.ErrNoElement {
			goto doRead // continue reading
		}
		s.handleSessionResult(elem, sErr)

	doRead:
		tm := time.AfterFunc(s.cfg.keepAliveTimeout, s.connTimeout) // schedule read timeout
		elem, sErr = s.session.Receive()
		tm.Stop()
	}
}

func (s *inS2S) handleSessionResult(elem stravaganza.Element, sErr error) {
	doneCh := make(chan struct{})
	s.rq.Run(func() {
		defer close(doneCh)

		ctx, cancel := s.requestContext()
		defer cancel()

		switch {
		case sErr == nil && elem != nil:
			err := s.handleElement(ctx, elem)
			if err != nil {
				log.Warnw("failed to process incoming S2S session element", "error", err, "id", s.id)
				return
			}

		case sErr != nil:
			s.handleSessionError(ctx, sErr)
		}
	})
	<-doneCh
}

func (s *inS2S) connTimeout() {
	s.rq.Run(func() {
		ctx, cancel := s.requestContext()
		defer cancel()
		_ = s.disconnect(ctx, streamerror.E(streamerror.ConnectionTimeout))
	})
}

func (s *inS2S) handleElement(ctx context.Context, elem stravaganza.Element) error {
	var err error
	t0 := time.Now()
	switch s.getState() {
	case inConnecting:
		err = s.handleConnecting(ctx, elem)
	case inConnected:
		err = s.handleConnected(ctx, elem)
	default:
		break
	}
	reportIncomingRequest(
		elem.Name(),
		elem.Attribute(stravaganza.Type),
		time.Since(t0).Seconds(),
	)
	return err
}

func (s *inS2S) handleConnecting(ctx context.Context, elem stravaganza.Element) error {
	// open stream session
	s.target = elem.Attribute(stravaganza.To)
	if len(s.target) == 0 {
		s.target = s.hosts.DefaultHostName()
	}
	s.sender = elem.Attribute(stravaganza.From)

	// set remote domain JID
	s.jd, _ = jid.New("", s.sender, "", true)
	s.session.SetFromJID(s.jd)

	fb := stravaganza.NewBuilder("stream:features")
	fb.WithAttribute("xmlns:stream", streamNamespace)
	fb.WithAttribute("version", "1.0")

	if !s.flags.isSecured() {
		fb.WithChild(stravaganza.NewBuilder("starttls").
			WithAttribute(stravaganza.Namespace, tlsNamespace).
			WithChild(
				stravaganza.NewBuilder("required").
					Build(),
			).
			Build(),
		)
		s.setState(inConnected)
		if err := s.session.OpenStream(ctx); err != nil {
			return err
		}
		return s.session.Send(ctx, fb.Build())
	}
	if !s.flags.isAuthenticated() {
		fb.WithChild(stravaganza.NewBuilder("mechanisms").
			WithAttribute(stravaganza.Namespace, saslNamespace).
			WithChild(
				stravaganza.NewBuilder("mechanism").
					WithText("EXTERNAL").
					Build(),
			).
			Build(),
		)
	}
	fb.WithChild(stravaganza.NewBuilder("dialback").
		WithAttribute(stravaganza.Namespace, dialbackNamespace).
		Build(),
	)
	s.setState(inConnected)
	if err := s.session.OpenStream(ctx); err != nil {
		return err
	}
	return s.session.Send(ctx, fb.Build())
}

func (s *inS2S) handleConnected(ctx context.Context, elem stravaganza.Element) error {
	if !s.flags.isSecured() {
		return s.proceedStartTLS(ctx, elem)
	}
	switch {
	case elem.Name() == "auth" && !s.flags.isAuthenticated():
		return s.authenticate(ctx, elem)

	case elem.Name() == "db:result" && !s.flags.isDialbackKeyAuthorized():
		return s.authorizeDialbackKey(ctx, elem)

	case elem.Name() == "db:verify":
		return s.verifyDialbackKey(ctx, elem)

	default:
		if s.flags.isAuthenticated() || s.flags.isDialbackKeyAuthorized() {
			// post element received event
			hInf := &hook.S2SStreamInfo{
				ID:      s.ID().String(),
				Sender:  s.sender,
				Target:  s.target,
				Element: elem,
			}
			halted, err := s.runHook(ctx, hook.S2SInStreamElementReceived, hInf)
			if err != nil {
				return err
			}
			if halted {
				return nil
			}
			switch stanza := hInf.Element.(type) {
			case stravaganza.Stanza:
				return s.processStanza(ctx, stanza)

			default:
				return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
			}
		}
		return nil
	}
}

func (s *inS2S) processStanza(ctx context.Context, stanza stravaganza.Stanza) error {
	toJID := stanza.ToJID()
	if s.comps.IsComponentHost(toJID.Domain()) {
		return s.comps.ProcessStanza(ctx, stanza)
	}
	// handle stanza
	switch stz := stanza.(type) {
	case *stravaganza.IQ:
		return s.processIQ(ctx, stz)
	case *stravaganza.Presence:
		return s.processPresence(ctx, stz)
	case *stravaganza.Message:
		return s.processMessage(ctx, stz)
	default:
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	}
}

func (s *inS2S) processIQ(ctx context.Context, iq *stravaganza.IQ) error {
	// run IQ received hook
	_, err := s.runHook(ctx, hook.S2SInStreamIQReceived, &hook.S2SStreamInfo{
		ID:      s.ID().String(),
		Sender:  s.sender,
		Target:  s.target,
		Element: iq,
	})
	if err != nil {
		return err
	}
	if iq.IsResult() || iq.IsError() {
		return nil // silently ignore
	}
	if s.mods.IsModuleIQ(iq) {
		return s.mods.ProcessIQ(ctx, iq)
	}
	// run will route iq hook
	hInf := &hook.S2SStreamInfo{
		ID:      s.ID().String(),
		Sender:  s.sender,
		Target:  s.target,
		Element: iq,
	}
	halted, err := s.runHook(ctx, hook.S2SInStreamWillRouteElement, hInf)
	if halted {
		return nil
	}
	if err != nil {
		return err
	}
	outIQ, ok := hInf.Element.(*stravaganza.IQ)
	if !ok {
		return nil
	}
	_, err = s.router.Route(ctx, outIQ)
	switch err {
	case router.ErrResourceNotFound:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.ServiceUnavailable, iq).Element())

	case router.ErrRemoteServerNotFound:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerNotFound, iq).Element())

	case router.ErrRemoteServerTimeout:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerTimeout, iq).Element())

	case nil:
		_, err = s.runHook(ctx, hook.S2SInStreamIQRouted, &hook.S2SStreamInfo{
			ID:      s.ID().String(),
			Sender:  s.sender,
			Target:  s.target,
			Element: iq,
		})
		return err
	}
	return nil
}

func (s *inS2S) processMessage(ctx context.Context, message *stravaganza.Message) error {
	// post message received event
	_, err := s.runHook(ctx, hook.S2SInStreamMessageReceived, &hook.S2SStreamInfo{
		ID:      s.ID().String(),
		Sender:  s.sender,
		Target:  s.target,
		Element: message,
	})
	if err != nil {
		return err
	}
	msg := message

sendMsg:
	// run will route Message hook
	hInf := &hook.S2SStreamInfo{
		ID:      s.ID().String(),
		Sender:  s.sender,
		Target:  s.target,
		Element: msg,
	}
	halted, err := s.runHook(ctx, hook.S2SInStreamWillRouteElement, hInf)
	if halted {
		return nil
	}
	if err != nil {
		return err
	}
	outMsg, ok := hInf.Element.(*stravaganza.Message)
	if !ok {
		return nil
	}
	_, err = s.router.Route(ctx, outMsg)
	switch err {
	case router.ErrResourceNotFound:
		// treat the stanza as if it were addressed to <node@domain>
		msg, _ = stravaganza.NewBuilderFromElement(msg).
			WithAttribute(stravaganza.From, message.FromJID().String()).
			WithAttribute(stravaganza.To, message.ToJID().ToBareJID().String()).
			BuildMessage()
		goto sendMsg

	case router.ErrNotExistingAccount:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.ServiceUnavailable, message).Element())

	case router.ErrRemoteServerNotFound:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerNotFound, message).Element())

	case router.ErrRemoteServerTimeout:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerTimeout, message).Element())

	case router.ErrUserNotAvailable:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.ServiceUnavailable, message).Element())

	case nil:
		_, err = s.runHook(ctx, hook.S2SInStreamMessageRouted, &hook.S2SStreamInfo{
			ID:      s.ID().String(),
			Sender:  s.sender,
			Target:  s.target,
			Element: msg,
		})
		return err
	}
	return nil
}

func (s *inS2S) processPresence(ctx context.Context, presence *stravaganza.Presence) error {
	// run presence received hook
	_, err := s.runHook(ctx, hook.S2SInStreamPresenceReceived, &hook.S2SStreamInfo{
		ID:      s.ID().String(),
		Sender:  s.sender,
		Target:  s.target,
		Element: presence,
	})
	if err != nil {
		return err
	}
	if presence.ToJID().IsFullWithUser() {
		// run will route presence hook
		hInf := &hook.S2SStreamInfo{
			ID:      s.ID().String(),
			Sender:  s.sender,
			Target:  s.target,
			Element: presence,
		}
		halted, err := s.runHook(ctx, hook.S2SInStreamWillRouteElement, hInf)
		if halted {
			return nil
		}
		if err != nil {
			return err
		}
		outPr, ok := hInf.Element.(*stravaganza.Presence)
		if !ok {
			return nil
		}
		_, err = s.router.Route(ctx, outPr)
		switch err {
		case nil:
			_, err := s.runHook(ctx, hook.S2SInStreamPresenceRouted, &hook.S2SStreamInfo{
				ID:      s.ID().String(),
				Sender:  s.sender,
				Target:  s.target,
				Element: presence,
			})
			return err
		}
		return nil
	}
	return nil
}

func (s *inS2S) authenticate(ctx context.Context, elem stravaganza.Element) error {
	if elem.Attribute(stravaganza.Namespace) != saslNamespace {
		return s.disconnect(ctx, streamerror.E(streamerror.InvalidNamespace))
	}
	if elem.Attribute("mechanism") != "EXTERNAL" {
		return s.failAuthentication(ctx, "invalid-mechanism", "")
	}
	// validate initiating server certificate
	certs := s.tr.PeerCertificates()
	for _, cert := range certs {
		for _, dnsName := range cert.DNSNames {
			if dnsName == s.sender {
				return s.finishAuthentication(ctx)
			}
		}
	}
	return s.failAuthentication(ctx, "bad-protocol", "Failed to get peer certificate")
}

func (s *inS2S) failAuthentication(ctx context.Context, reason, text string) error {
	log.Infow("failed S2S incoming stream authentication",
		"id", s.id,
		"sender", s.sender,
		"target", s.target,
		"reason", reason,
	)
	sb := stravaganza.NewBuilder("failure").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		WithChild(stravaganza.NewBuilder(reason).Build())
	if len(text) > 0 {
		sb.WithChild(
			stravaganza.NewBuilder("text").
				WithText(text).
				Build(),
		)
	}
	return s.sendElement(ctx, sb.Build())
}

func (s *inS2S) finishAuthentication(ctx context.Context) error {
	// update rate limiter
	if err := s.updateRateLimiter(); err != nil {
		return err
	}
	log.Infow("authenticated S2S incoming stream",
		"id", s.id,
		"sender", s.sender,
		"target", s.target)

	s.flags.setAuthenticated()
	s.restartSession()

	return s.sendElement(ctx, stravaganza.NewBuilder("success").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		Build(),
	)
}

func (s *inS2S) authorizeDialbackKey(ctx context.Context, elem stravaganza.Element) error {
	if !s.hosts.IsLocalHost(elem.Attribute(stravaganza.To)) {
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.ItemNotFound, elem).Element())
	}
	elemFrom := elem.Attribute(stravaganza.From)
	elemTo := elem.Attribute(stravaganza.To)

	dbParams := DialbackParams{
		StreamID: s.session.StreamID(),
		From:     elemTo,
		To:       elemFrom,
		Key:      elem.Text(),
	}
	dbOut, err := s.outProvider.GetDialback(ctx, s.hosts.DefaultHostName(), elem.Attribute(stravaganza.From), dbParams)
	if err != nil {
		log.Errorf("Failed to obtain S2S dialback connection: %v", err)
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerTimeout, elem).Element())
	}
	go func() {
		dbRes := <-dbOut.DialbackResult()
		s.rq.Run(func() {
			ctx, cancel := s.requestContext()
			defer cancel()

			err := s.handleDialbackResult(ctx, elemFrom, elemTo, dbRes)
			if err != nil {
				log.Errorf("Failed to process S2S dialback response: %v", err)
			}
		})
	}()
	s.setState(inAuthorizingDialbackKey)
	return nil
}

func (s *inS2S) handleDialbackResult(ctx context.Context, from, to string, dbRes stream.DialbackResult) error {
	sb := stravaganza.NewBuilder("db:result")
	sb.WithAttribute(stravaganza.From, to)
	sb.WithAttribute(stravaganza.To, from)
	if dbRes.Valid {
		sb.WithAttribute(stravaganza.Type, "valid")

		// update rate limiter
		if err := s.updateRateLimiter(); err != nil {
			return err
		}
		log.Infow("authorized S2S dialback key",
			"id", s.id,
			"sender", s.sender,
			"target", s.target,
		)
		s.flags.setDialbackKeyAuthorized()
	} else {
		sb.WithAttribute(stravaganza.Type, "invalid")
	}
	s.setState(inConnected)
	return s.sendElement(ctx, sb.Build())
}

func (s *inS2S) verifyDialbackKey(ctx context.Context, elem stravaganza.Element) error {
	if !s.hosts.IsLocalHost(elem.Attribute(stravaganza.To)) {
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.ItemNotFound, elem).Element())
	}
	sender := elem.Attribute(stravaganza.From)
	target := elem.Attribute(stravaganza.To)
	streamID := elem.Attribute(stravaganza.ID)

	sb := stravaganza.NewBuilder("db:verify").
		WithAttribute(stravaganza.From, target).
		WithAttribute(stravaganza.To, sender).
		WithAttribute(stravaganza.ID, streamID)

	// check whether we have an active dialback request
	dbReqOn, err := isDbRequestOn(ctx, sender, target, streamID, s.kv)
	if err != nil {
		return err
	}
	expectedKey := dbKey(
		s.outProvider.DialbackSecret(),
		sender,
		target,
		streamID,
	)
	if dbReqOn && expectedKey == elem.Text() {
		// unregister dialback request
		if err := unregisterDbRequest(ctx, streamID, s.kv); err != nil {
			return err
		}
		log.Infow("S2S dialback key successfully verified",
			"id", s.id,
			"sender", s.sender,
			"target", s.target,
			"key", elem.Text(),
		)
		sb.WithAttribute("type", "valid")

	} else {
		log.Infow("failed to verify S2S dialback key",
			"id", s.id,
			"sender", s.sender,
			"target", s.target,
			"expected_key", expectedKey,
			"key", elem.Text(),
		)
		sb.WithAttribute("type", "invalid")
	}
	return s.sendElement(ctx, sb.Build())
}

func (s *inS2S) proceedStartTLS(ctx context.Context, elem stravaganza.Element) error {
	if elem.Attribute(stravaganza.Namespace) != tlsNamespace {
		return s.disconnect(ctx, streamerror.E(streamerror.InvalidNamespace))
	} else if elem.Name() != "starttls" {
		return s.disconnect(ctx, streamerror.E(streamerror.NotAuthorized))
	}
	err := s.sendElement(ctx, stravaganza.NewBuilder("proceed").
		WithAttribute(stravaganza.Namespace, tlsNamespace).
		Build(),
	)
	if err != nil {
		return err
	}
	s.tr.StartTLS(&tls.Config{
		ServerName:   s.target,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		Certificates: s.hosts.Certificates(),
	}, false)
	s.flags.setSecured()

	log.Infow("secured S2S incoming stream", "id", s.id, "sender", s.sender, "target", s.target)

	s.restartSession()
	return nil
}

func (s *inS2S) handleSessionError(ctx context.Context, err error) {
	switch err {
	case xmppparser.ErrStreamClosedByPeer:
		_ = s.session.Close(ctx)
		fallthrough

	default:
		_ = s.close(ctx)
	}
}

func (s *inS2S) restartSession() {
	_ = s.session.Reset(s.tr)
	s.setState(inConnecting)
}

func (s *inS2S) updateRateLimiter() error {
	rLim := s.shapers.MatchingJID(s.jd).RateLimiter()
	return s.tr.SetReadRateLimiter(rLim)
}

func (s *inS2S) disconnect(ctx context.Context, streamErr *streamerror.Error) error {
	if s.getState() == inConnecting {
		_ = s.session.OpenStream(ctx)
	}
	if streamErr != nil {
		if err := s.sendElement(ctx, streamErr.Element()); err != nil {
			return err
		}
	}
	// close stream session and wait for the other entity to close its stream
	_ = s.session.Close(ctx)

	if s.getState() != inConnecting && streamErr != nil && streamErr.Reason == streamerror.ConnectionTimeout {
		s.discTm = time.AfterFunc(inDisconnectTimeout, func() {
			s.rq.Run(func() {
				ctx, cancel := s.requestContext()
				defer cancel()
				_ = s.close(ctx)
			})
		})
		s.sendDisabled = true // avoid sending anymore stanzas while closing
		return nil
	}
	return s.close(ctx)
}

func (s *inS2S) close(ctx context.Context) error {
	if s.getState() == inDisconnected {
		return nil // already disconnected
	}
	defer close(s.doneCh)

	s.setState(inDisconnected)

	if s.discTm != nil {
		s.discTm.Stop()
	}
	// unregister S2S stream
	s.inHub.unregister(s)

	log.Infow("unregistered S2S incoming stream",
		"id", s.id,
		"sender", s.sender,
		"target", s.target,
	)
	// run unregistered incoming S2S hook
	_, err := s.runHook(ctx, hook.S2SInStreamUnregistered, &hook.S2SStreamInfo{
		ID: s.ID().String(),
	})
	if err != nil {
		return err
	}
	reportIncomingConnectionUnregistered()

	// close underlying transport
	_ = s.tr.Close()
	return nil
}

func (s *inS2S) sendElement(ctx context.Context, elem stravaganza.Element) error {
	if s.sendDisabled {
		return nil
	}
	var err error
	err = s.session.Send(ctx, elem)
	reportOutgoingRequest(
		elem.Name(),
		elem.Attribute(stravaganza.Type),
	)
	return err
}

func (s *inS2S) setState(state inState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

func (s *inS2S) getState() inState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *inS2S) runHook(ctx context.Context, hookName string, inf *hook.S2SStreamInfo) (halt bool, err error) {
	return s.hk.Run(ctx, hookName, &hook.ExecutionContext{
		Info:   inf,
		Sender: s,
	})
}

func (s *inS2S) requestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.cfg.reqTimeout)
}

var currentID uint64

func nextStreamID() stream.S2SInID {
	return stream.S2SInID(atomic.AddUint64(&currentID, 1))
}
