// Copyright 2022 The jackal Authors
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

package c2s

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/jackal-xmpp/runqueue/v2"
	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/jackal-xmpp/stravaganza/jid"
	xmppparser "github.com/jackal-xmpp/stravaganza/parser"
	"github.com/ortuman/jackal/pkg/auth"
	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/ortuman/jackal/pkg/cluster/resourcemanager"
	"github.com/ortuman/jackal/pkg/component"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
	"github.com/ortuman/jackal/pkg/module"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
	xmppsession "github.com/ortuman/jackal/pkg/session"
	"github.com/ortuman/jackal/pkg/shaper"
	"github.com/ortuman/jackal/pkg/transport"
	"github.com/ortuman/jackal/pkg/transport/compress"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

type state uint32

const (
	inConnecting state = iota
	inConnected
	inAuthenticating
	inAuthenticated
	inBinded
	inDisconnected
	inTerminated
)

const (
	maxAuthFailed  = 5
	maxAuthAborted = 1
)

var (
	disconnectTimeout = time.Second * 5
)

type resourceConflict int8

const (
	override resourceConflict = iota
	disallow
	terminateOld
)

type inCfg struct {
	authenticateTimeout time.Duration
	reqTimeout          time.Duration
	maxStanzaSize       int
	compressionLevel    compress.Level
	resConflict         resourceConflict
	useTLS              bool
	tlsConfig           *tls.Config
}

type authState struct {
	authenticators []auth.Authenticator
	active         auth.Authenticator
	failedTimes    int
	abortTimes     int
}

func (a *authState) reset() {
	a.active.Reset()
	a.active = nil
}

type inC2S struct {
	id           stream.C2SID
	cfg          inCfg
	tr           transport.Transport
	authSt       authState
	hosts        hosts
	router       router.Router
	comps        components
	mods         modules
	resMng       resourcemanager.Manager
	session      session
	shapers      shaper.Shapers
	hk           *hook.Hooks
	logger       kitlog.Logger
	rq           *runqueue.RunQueue
	discTm       *time.Timer
	doneCh       chan struct{}
	sendDisabled bool

	mu    sync.RWMutex
	state state
	jd    *jid.JID
	pr    *stravaganza.Presence
	inf   *c2smodel.InfoMap
	flags flags
}

func newInC2S(
	cfg inCfg,
	tr transport.Transport,
	authenticators []auth.Authenticator,
	hosts *host.Hosts,
	router router.Router,
	comps *component.Components,
	mods *module.Modules,
	resMng resourcemanager.Manager,
	shapers shaper.Shapers,
	hk *hook.Hooks,
	logger kitlog.Logger,
) (*inC2S, error) {
	// set default rate limiter
	rLim := shapers.DefaultC2S().RateLimiter()
	if err := tr.SetReadRateLimiter(rLim); err != nil {
		return nil, err
	}
	// create session
	id := nextStreamID()

	sLogger := kitlog.With(logger, "id", id)
	session := xmppsession.New(
		xmppsession.C2SSession,
		id.String(),
		tr,
		hosts,
		xmppsession.Config{
			MaxStanzaSize: cfg.maxStanzaSize,
		},
		sLogger,
	)
	// init stream
	stm := &inC2S{
		id:      id,
		cfg:     cfg,
		tr:      tr,
		inf:     c2smodel.NewInfoMap(),
		session: session,
		authSt:  authState{authenticators: authenticators},
		hosts:   hosts,
		router:  router,
		comps:   comps,
		mods:    mods,
		resMng:  resMng,
		shapers: shapers,
		rq:      runqueue.New(id.String()),
		doneCh:  make(chan struct{}),
		state:   inConnecting,
		hk:      hk,
		logger:  sLogger,
	}
	if cfg.useTLS {
		stm.flags.setSecured() // stream already secured
	}
	return stm, nil
}

func (s *inC2S) ID() stream.C2SID {
	return s.id
}

func (s *inC2S) SetInfoValue(ctx context.Context, k string, val interface{}) error {
	s.mu.Lock()
	switch v := val.(type) {
	case string:
		s.inf.SetString(k, v)
	case bool:
		s.inf.SetBool(k, v)
	case int:
		s.inf.SetInt(k, v)
	case float64:
		s.inf.SetFloat(k, v)
	default:
		s.mu.Unlock()
		return fmt.Errorf("c2s: unsupported info value: %T", val)
	}
	s.mu.Unlock()

	return s.resMng.PutResource(ctx, s.getResource())
}

func (s *inC2S) Info() c2smodel.Info {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inf
}

func (s *inC2S) JID() *jid.JID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jd
}

func (s *inC2S) Username() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if jd := s.jd; jd != nil {
		return jd.Node()
	}
	return ""
}

func (s *inC2S) Domain() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if jd := s.jd; jd != nil {
		return jd.Domain()
	}
	return ""
}

func (s *inC2S) Resource() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if jd := s.jd; jd != nil {
		return jd.Resource()
	}
	return ""
}

func (s *inC2S) IsSecured() bool {
	return s.flags.isSecured()
}

func (s *inC2S) IsAuthenticated() bool {
	return s.flags.isAuthenticated()
}

func (s *inC2S) IsBinded() bool {
	return s.flags.isBinded()
}

func (s *inC2S) Presence() *stravaganza.Presence {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pr
}

func (s *inC2S) SendElement(elem stravaganza.Element) <-chan error {
	errCh := make(chan error, 1)
	s.rq.Run(func() {
		ctx, cancel := s.requestContext()
		defer cancel()
		errCh <- s.sendElement(ctx, elem)
	})
	return errCh
}

func (s *inC2S) Disconnect(streamErr *streamerror.Error) <-chan error {
	errCh := make(chan error, 1)
	s.rq.Run(func() {
		ctx, cancel := s.requestContext()
		defer cancel()
		errCh <- s.disconnect(ctx, streamErr)
	})
	return errCh
}

func (s *inC2S) Resume(ctx context.Context, jd *jid.JID, pr *stravaganza.Presence, inf c2smodel.Info) error {
	s.mu.Lock()
	s.jd = jd
	s.pr = pr
	s.inf = c2smodel.NewInfoMapFromInfo(inf)
	s.mu.Unlock()

	s.session.SetFromJID(jd)

	if err := s.bindC2S(ctx); err != nil {
		return err
	}
	s.setState(inBinded)
	s.flags.setBinded()

	// run binded C2S hook
	_, err := s.runHook(ctx, hook.C2SStreamBinded, &hook.C2SStreamInfo{
		ID:  s.ID().String(),
		JID: s.JID(),
	})
	return err
}

func (s *inC2S) bindC2S(ctx context.Context) error {
	// update rate limiter
	if err := s.updateRateLimiter(); err != nil {
		return err
	}
	// bind and register cluster resource
	if err := s.router.C2S().Bind(s.ID()); err != nil {
		return err
	}
	return s.resMng.PutResource(ctx, s.getResource())
}

func (s *inC2S) Done() <-chan struct{} {
	return s.doneCh
}

func (s *inC2S) start() error {
	// register C2S stream
	if err := s.router.C2S().Register(s); err != nil {
		return err
	}
	// run registered C2S hook
	ctx, cancel := s.requestContext()
	_, err := s.runHook(ctx, hook.C2SStreamConnected, &hook.C2SStreamInfo{
		ID: s.ID().String(),
	})
	cancel()

	if err != nil {
		return err
	}
	reportConnectionRegistered()

	s.readLoop()
	return nil
}

func (s *inC2S) readLoop() {
	s.restartSession()

	s.tr.SetConnectDeadlineHandler(s.connTimeout)
	s.tr.SetKeepAliveDeadlineHandler(s.connTimeout)

	// schedule authenticate timeout
	authTm := time.AfterFunc(s.cfg.authenticateTimeout, s.connTimeout)
	defer authTm.Stop()

	for {
		elem, sErr := s.session.Receive()

		// process result and update state accordingly
		s.handleSessionResult(elem, sErr)

		switch s.getState() {
		case inAuthenticated:
			authTm.Stop()
		case inDisconnected, inTerminated:
			return
		}
	}
}

func (s *inC2S) handleSessionResult(elem stravaganza.Element, sErr error) {
	handledCh := make(chan struct{})
	s.rq.Run(func() {
		defer close(handledCh)

		ctx, cancel := s.requestContext()
		defer cancel()

		switch {
		case sErr == nil && elem != nil:
			err := s.handleElement(ctx, elem)
			if err != nil {
				level.Warn(s.logger).Log("msg", "failed to process incoming C2S session element", "err", err)
				return
			}

		case sErr != nil:
			s.handleSessionError(ctx, sErr)
		}
	})
	<-handledCh
}

func (s *inC2S) connTimeout() {
	s.rq.Run(func() {
		ctx, cancel := s.requestContext()
		defer cancel()
		_ = s.disconnect(ctx, streamerror.E(streamerror.ConnectionTimeout))
	})
}

func (s *inC2S) handleElement(ctx context.Context, elem stravaganza.Element) error {
	// run received element hook
	hi := &hook.C2SStreamInfo{
		ID:       s.ID().String(),
		JID:      s.JID(),
		Presence: s.Presence(),
		Element:  elem,
	}
	halted, err := s.runHook(ctx, hook.C2SStreamElementReceived, hi)
	if halted {
		return nil
	}
	if err != nil {
		return err
	}

	t0 := time.Now()
	switch s.getState() {
	case inConnecting:
		err = s.handleConnecting(ctx, hi.Element)
	case inConnected:
		err = s.handleConnected(ctx, hi.Element)
	case inAuthenticating:
		err = s.handleAuthenticating(ctx, hi.Element)
	case inAuthenticated:
		err = s.handleAuthenticated(ctx, hi.Element)
	case inBinded:
		err = s.handleBinded(ctx, hi.Element)
	}
	reportIncomingRequest(
		elem.Name(),
		elem.Attribute(stravaganza.Type),
		time.Since(t0).Seconds(),
	)
	return err
}

func (s *inC2S) handleConnecting(ctx context.Context, elem stravaganza.Element) error {
	// assign stream domain if not set yet
	if len(s.Domain()) == 0 {
		j, _ := jid.NewWithString(elem.Attribute(stravaganza.To), true)
		s.setJID(j)
	}

	// open stream session
	s.session.SetFromJID(s.JID())

	fb := stravaganza.NewBuilder("stream:features").
		WithAttribute(stravaganza.StreamNamespace, streamNamespace).
		WithAttribute(stravaganza.Version, "1.0")

	if !s.flags.isAuthenticated() {
		fb.WithChildren(s.unauthenticatedFeatures()...)
		s.setState(inConnected)
	} else {
		authFeatures, err := s.authenticatedFeatures(ctx)
		if err != nil {
			return err
		}
		fb.WithChildren(authFeatures...)
		s.setState(inAuthenticated)
	}
	if err := s.session.OpenStream(ctx); err != nil {
		return err
	}
	return s.session.Send(ctx, fb.Build())
}

func (s *inC2S) handleConnected(ctx context.Context, elem stravaganza.Element) error {
	switch elem.Name() {
	case "starttls":
		return s.proceedStartTLS(ctx, elem)

	case "auth":
		return s.startAuthentication(ctx, elem)

	case "iq":
		if elem.ChildNamespace("query", "jabber:iq:auth") != nil {
			// do not allow non-SASL authentication
			return s.sendElement(ctx, stanzaerror.E(stanzaerror.ServiceUnavailable, elem).Element())
		}
		fallthrough

	case "message", "presence":
		return s.disconnect(ctx, streamerror.E(streamerror.NotAuthorized))

	default:
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	}
}

func (s *inC2S) handleAuthenticating(ctx context.Context, elem stravaganza.Element) error {
	if elem.Attribute(stravaganza.Namespace) != saslNamespace {
		return s.disconnect(ctx, streamerror.E(streamerror.InvalidNamespace))
	}
	if elem.Name() == "abort" { // initiating entity aborted the handshake
		return s.abortAuthentication(ctx)
	}
	if err := s.continueAuthentication(ctx, elem); err != nil {
		if saslErr, ok := err.(*auth.SASLError); ok {
			return s.failAuthentication(ctx, saslErr)
		}
		return err
	}
	if s.authSt.active.Authenticated() {
		return s.finishAuthentication()
	}
	return nil
}

func (s *inC2S) handleAuthenticated(ctx context.Context, elem stravaganza.Element) error {
	switch elem.Name() {
	case "compress":
		return s.compress(ctx, elem)
	case "iq":
		return s.bindResource(ctx, elem.(*stravaganza.IQ))
	default:
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	}
}

func (s *inC2S) handleBinded(ctx context.Context, elem stravaganza.Element) error {
	switch stanza := elem.(type) {
	case stravaganza.Stanza:
		return s.processStanza(ctx, stanza)

	default:
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	}
}

func (s *inC2S) processStanza(ctx context.Context, stanza stravaganza.Stanza) error {
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

func (s *inC2S) processIQ(ctx context.Context, iq *stravaganza.IQ) error {
	// run iq received hook
	hi := &hook.C2SStreamInfo{
		ID:       s.ID().String(),
		JID:      s.JID(),
		Presence: s.Presence(),
		Element:  iq,
	}
	_, err := s.runHook(ctx, hook.C2SStreamIQReceived, hi)
	if err != nil {
		return err
	}
	iq = hi.Element.(*stravaganza.IQ)

	if iq.IsSet() && iq.ChildNamespace("session", sessionNamespace) != nil {
		if !s.flags.isSessionStarted() {
			s.flags.setSessionStarted()
			return s.sendElement(ctx, iq.ResultBuilder().Build())
		}
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.NotAllowed, iq).Element())
	}
	if iq.IsResult() || iq.IsError() {
		return nil // silently ignore
	}
	if s.mods.IsModuleIQ(iq) {
		return s.mods.ProcessIQ(ctx, iq)
	}
	// run will route iq hook
	hi = &hook.C2SStreamInfo{
		ID:       s.ID().String(),
		JID:      s.JID(),
		Presence: s.Presence(),
		Element:  iq,
	}
	halted, err := s.runHook(ctx, hook.C2SStreamWillRouteElement, hi)
	if halted {
		return nil
	}
	if err != nil {
		return err
	}
	iq = hi.Element.(*stravaganza.IQ)

	targets, err := s.router.Route(ctx, iq)
	switch err {
	case router.ErrResourceNotFound:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.ServiceUnavailable, iq).Element())

	case router.ErrRemoteServerNotFound:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerNotFound, iq).Element())

	case router.ErrRemoteServerTimeout:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerTimeout, iq).Element())

	case nil, router.ErrUserNotAvailable:
		_, err = s.runHook(ctx, hook.C2SStreamIQRouted, &hook.C2SStreamInfo{
			ID:       s.ID().String(),
			JID:      s.JID(),
			Presence: s.Presence(),
			Targets:  targets,
			Element:  iq,
		})
		return err
	}
	return nil
}

func (s *inC2S) processPresence(ctx context.Context, presence *stravaganza.Presence) error {
	// run presence received hook
	hi := &hook.C2SStreamInfo{
		ID:       s.ID().String(),
		JID:      s.JID(),
		Presence: s.Presence(),
		Element:  presence,
	}
	_, err := s.runHook(ctx, hook.C2SStreamPresenceReceived, hi)
	if err != nil {
		return err
	}
	presence = hi.Element.(*stravaganza.Presence)

	if presence.ToJID().IsFullWithUser() {
		// run will route presence hook
		hi = &hook.C2SStreamInfo{
			ID:       s.ID().String(),
			JID:      s.JID(),
			Presence: s.Presence(),
			Element:  presence,
		}
		halted, err := s.runHook(ctx, hook.C2SStreamWillRouteElement, hi)
		if halted {
			return nil
		}
		if err != nil {
			return err
		}
		presence = hi.Element.(*stravaganza.Presence)

		targets, err := s.router.Route(ctx, presence)
		switch err {
		case nil, router.ErrUserNotAvailable:
			_, err = s.runHook(ctx, hook.C2SStreamPresenceRouted, &hook.C2SStreamInfo{
				ID:      s.ID().String(),
				JID:     s.JID(),
				Targets: targets,
				Element: presence,
			})
			return err
		}
		return nil
	}
	// update presence
	matchesUserJID := s.JID().MatchesWithOptions(presence.ToJID(), jid.MatchesBare)
	if matchesUserJID && (presence.IsAvailable() || presence.IsUnavailable()) {
		s.setPresence(presence)
	}
	// update cluster resource
	return s.resMng.PutResource(ctx, s.getResource())
}

func (s *inC2S) processMessage(ctx context.Context, message *stravaganza.Message) error {
	// run message received hook
	hi := &hook.C2SStreamInfo{
		ID:       s.ID().String(),
		JID:      s.JID(),
		Presence: s.Presence(),
		Element:  message,
	}
	_, err := s.runHook(ctx, hook.C2SStreamMessageReceived, hi)
	if err != nil {
		return err
	}
	msg := hi.Element.(*stravaganza.Message)

sendMsg:
	// run will route message hook
	hi = &hook.C2SStreamInfo{
		ID:       s.ID().String(),
		JID:      s.JID(),
		Presence: s.Presence(),
		Element:  msg,
	}
	halted, err := s.runHook(ctx, hook.C2SStreamWillRouteElement, hi)
	if halted {
		return nil
	}
	if err != nil {
		return err
	}
	msg = hi.Element.(*stravaganza.Message)

	targets, err := s.router.Route(ctx, msg)
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

	case nil, router.ErrUserNotAvailable:
		halted, hErr := s.runHook(ctx, hook.C2SStreamMessageRouted, &hook.C2SStreamInfo{
			ID:       s.ID().String(),
			JID:      s.JID(),
			Presence: s.Presence(),
			Targets:  targets,
			Element:  msg,
		})
		if halted {
			return nil
		}
		if errors.Is(err, router.ErrUserNotAvailable) {
			return s.sendElement(ctx, stanzaerror.E(stanzaerror.ServiceUnavailable, message).Element())
		}
		return hErr

	default:
		return err
	}
}

func (s *inC2S) handleSessionError(ctx context.Context, err error) {
	if errors.Is(err, xmppparser.ErrStreamClosedByPeer) {
		_ = s.session.Close(ctx)
	}
	_ = s.close(ctx, err)
}

func (s *inC2S) unauthenticatedFeatures() []stravaganza.Element {
	var features []stravaganza.Element

	// attach start-tls feature
	isSocketTr := s.tr.Type() == transport.Socket
	if isSocketTr && !s.flags.isSecured() {
		features = append(features, stravaganza.NewBuilder("starttls").
			WithAttribute(stravaganza.Namespace, "urn:ietf:params:xml:ns:xmpp-tls").
			WithChild(stravaganza.NewBuilder("required").Build()).
			Build(),
		)
	}
	// attach SASL mechanisms
	shouldOfferSASL := !isSocketTr || (isSocketTr && s.flags.isSecured())

	if shouldOfferSASL && len(s.authSt.authenticators) > 0 {
		supportsCb := s.tr.SupportsChannelBinding()

		sb := stravaganza.NewBuilder("mechanisms")
		sb.WithAttribute(stravaganza.Namespace, saslNamespace)
		for _, authenticator := range s.authSt.authenticators {
			if authenticator.UsesChannelBinding() && !supportsCb {
				continue // transport doesn't support channel binding
			}
			sb.WithChild(
				stravaganza.NewBuilder("mechanism").
					WithText(authenticator.Mechanism()).
					Build(),
			)
		}
		features = append(features, sb.Build())
	}
	return features
}

func (s *inC2S) authenticatedFeatures(ctx context.Context) ([]stravaganza.Element, error) {
	var features []stravaganza.Element

	isSocketTr := s.tr.Type() == transport.Socket

	// compression feature
	compressionAvailable := isSocketTr && s.cfg.compressionLevel != compress.NoCompression

	if !s.flags.isCompressed() && compressionAvailable {
		compressionElem := stravaganza.NewBuilder("compression").
			WithAttribute(stravaganza.Namespace, "http://jabber.org/features/compress").
			WithChild(
				stravaganza.NewBuilder("method").
					WithText("zlib").
					Build(),
			).
			Build()
		features = append(features, compressionElem)
	}
	// bind feature
	bindElem := stravaganza.NewBuilder("bind").
		WithAttribute(stravaganza.Namespace, "urn:ietf:params:xml:ns:xmpp-bind").
		WithChild(stravaganza.NewBuilder("required").Build()).
		Build()
	features = append(features, bindElem)

	// [rfc6121] offer session feature for backward compatibility
	sessElem := stravaganza.NewBuilder("session").
		WithAttribute(stravaganza.Namespace, "urn:ietf:params:xml:ns:xmpp-session").
		Build()
	features = append(features, sessElem)

	// include module stream features
	modFeatures, err := s.mods.StreamFeatures(ctx, s.JID().Domain())
	if err != nil {
		return nil, err
	}
	return append(features, modFeatures...), nil
}

func (s *inC2S) proceedStartTLS(ctx context.Context, elem stravaganza.Element) error {
	if s.flags.isSecured() {
		return s.disconnect(ctx, streamerror.E(streamerror.NotAuthorized))
	}
	ns := elem.Attribute(stravaganza.Namespace)
	if len(ns) > 0 && ns != tlsNamespace {
		return s.disconnect(ctx, streamerror.E(streamerror.InvalidNamespace))
	}
	s.flags.setSecured()

	if err := s.sendElement(ctx,
		stravaganza.NewBuilder("proceed").
			WithAttribute(stravaganza.Namespace, tlsNamespace).
			Build(),
	); err != nil {
		return err
	}
	s.tr.StartTLS(&tls.Config{
		Certificates: s.hosts.Certificates(),
	}, false)

	level.Info(s.logger).Log("msg", "secured C2S stream")

	s.restartSession()
	return nil
}

func (s *inC2S) startAuthentication(ctx context.Context, elem stravaganza.Element) error {
	if elem.Attribute(stravaganza.Namespace) != saslNamespace {
		return s.disconnect(ctx, streamerror.E(streamerror.InvalidNamespace))
	}
	mechanism := elem.Attribute("mechanism")
	for _, authenticator := range s.authSt.authenticators {
		if authenticator.Mechanism() != mechanism {
			continue
		}
		s.authSt.active = authenticator
		if err := s.continueAuthentication(ctx, elem); err != nil {
			if saslErr, ok := err.(*auth.SASLError); ok {
				return s.failAuthentication(ctx, saslErr)
			}
			return err
		}
		if s.authSt.active.Authenticated() {
			return s.finishAuthentication()
		}
		s.setState(inAuthenticating)
		return nil
	}
	// ...mechanism not found...
	failureElem := stravaganza.NewBuilder("failure").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		WithChild(stravaganza.NewBuilder("invalid-mechanism").Build()).
		Build()
	return s.sendElement(ctx, failureElem)
}

func (s *inC2S) continueAuthentication(ctx context.Context, elem stravaganza.Element) error {
	elem, saslErr := s.authSt.active.ProcessElement(ctx, elem)
	if saslErr != nil {
		return saslErr
	}
	return s.sendElement(ctx, elem)
}

func (s *inC2S) finishAuthentication() error {
	username := s.authSt.active.Username()

	j, _ := jid.New(username, s.Domain(), "", true)
	s.setJID(j)
	s.flags.setAuthenticated()

	// update rate limiter
	if err := s.updateRateLimiter(); err != nil {
		return err
	}
	level.Info(s.logger).Log("msg", "authenticated C2S stream", "username", username)

	s.authSt.reset()
	s.restartSession()
	return nil
}

func (s *inC2S) failAuthentication(ctx context.Context, saslErr *auth.SASLError) error {
	if saslErr.Err != nil {
		level.Warn(s.logger).Log("msg", "authentication error", "err", saslErr.Err)
	}
	s.authSt.failedTimes++
	if s.authSt.failedTimes >= maxAuthFailed {
		return s.disconnect(ctx, streamerror.E(streamerror.PolicyViolation))
	}
	failureElem := stravaganza.NewBuilder("failure").
		WithAttribute(stravaganza.Namespace, saslNamespace).
		WithChild(saslErr.Element()).
		Build()
	return s.sendElement(ctx, failureElem)
}

func (s *inC2S) abortAuthentication(ctx context.Context) error {
	s.authSt.abortTimes++
	if s.authSt.abortTimes >= maxAuthAborted {
		return s.disconnect(ctx, streamerror.E(streamerror.PolicyViolation))
	}
	s.authSt.reset()
	s.setState(inConnected)
	return nil
}

func (s *inC2S) compress(ctx context.Context, elem stravaganza.Element) error {
	if elem.Attribute(stravaganza.Namespace) != compressNamespace || s.flags.isCompressed() {
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	}
	method := elem.Child("method")
	if method == nil || len(method.Text()) == 0 {
		failureElem := stravaganza.NewBuilder("failure").
			WithAttribute(stravaganza.Namespace, compressNamespace).
			WithChild(stravaganza.NewBuilder("setup-failed").Build()).
			Build()
		return s.sendElement(ctx, failureElem)
	}
	if method.Text() != "zlib" {
		failure := stravaganza.NewBuilder("failure").
			WithAttribute(stravaganza.Namespace, compressNamespace).
			WithChild(stravaganza.NewBuilder("unsupported-method").Build()).
			Build()
		return s.sendElement(ctx, failure)
	}
	if err := s.sendElement(ctx, stravaganza.NewBuilder("compressed").
		WithAttribute(stravaganza.Namespace, compressNamespace).
		Build(),
	); err != nil {
		return err
	}
	// compress transport
	s.tr.EnableCompression(s.cfg.compressionLevel)
	s.flags.setCompressed()

	level.Info(s.logger).Log("msg", "compressed C2S stream", "username", s.Username())

	s.restartSession()
	return nil
}

func (s *inC2S) bindResource(ctx context.Context, iq *stravaganza.IQ) error {
	bind := iq.ChildNamespace("bind", bindNamespace)
	if iq.Attribute(stravaganza.Type) != stravaganza.SetType || bind == nil {
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.NotAllowed, iq).Element())
	}
	// fetch active resources
	rss, err := s.resMng.GetResources(ctx, s.Username())
	if err != nil {
		return err
	}
	// check is max session count has been reached
	maxSessions := s.shapers.MatchingJID(s.JID()).MaxSessions
	if len(rss) == maxSessions {
		se := streamerror.E(streamerror.PolicyViolation)
		se.ApplicationElement = stravaganza.NewBuilder("reached-max-session-count").
			WithAttribute(stravaganza.Namespace, "urn:xmpp:errors").
			Build()
		return s.disconnect(ctx, se)
	}

	var res string
	if resElem := bind.Child("resource"); resElem != nil {
		res = resElem.Text()

		// check if another stream with same resource value did already connect
		for _, rs := range rss {
			if rs.JID().Resource() != res {
				continue
			}
			switch s.cfg.resConflict {
			// replace by a server generated resourcepart
			case override:
				res = uuid.New().String()
				break

			// disconnect previously connected resource
			case terminateOld:
				se := streamerror.E(streamerror.PolicyViolation)
				se.ApplicationElement = stravaganza.NewBuilder("resource-conflict").
					WithAttribute(stravaganza.Namespace, "urn:xmpp:errors").
					Build()
				if err := s.router.C2S().Disconnect(ctx, rs, se); err != nil {
					return err
				}
				break

			// disallow resource binding
			case disallow:
				return s.sendElement(ctx, stanzaerror.E(stanzaerror.Conflict, iq).Element())
			}
			break
		}
	} else {
		res = uuid.New().String() // server generated
	}

	// set stream jid and presence
	userJID, err := jid.New(s.Username(), s.Domain(), res, false)
	if err != nil {
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.BadRequest, iq).Element())
	}
	s.setJID(userJID)
	s.session.SetFromJID(userJID)

	pr, _ := stravaganza.NewPresenceBuilder().
		WithAttribute(stravaganza.From, userJID.String()).
		WithAttribute(stravaganza.To, userJID.String()).
		WithAttribute(stravaganza.Type, stravaganza.UnavailableType).
		BuildPresence()
	s.setPresence(pr)

	if err := s.bindC2S(ctx); err != nil {
		return err
	}
	s.setState(inBinded)
	s.flags.setBinded()

	// run binded C2S hook
	_, err = s.runHook(ctx, hook.C2SStreamBinded, &hook.C2SStreamInfo{
		ID:  s.ID().String(),
		JID: s.JID(),
	})
	if err != nil {
		return err
	}

	// notify successful binding
	resIQ := xmpputil.MakeResultIQ(iq,
		stravaganza.NewBuilder("bind").
			WithAttribute(stravaganza.Namespace, bindNamespace).
			WithChild(
				stravaganza.NewBuilder("jid").
					WithText(s.JID().String()).
					Build(),
			).
			Build(),
	)
	return s.sendElement(ctx, resIQ)
}

func (s *inC2S) disconnect(ctx context.Context, streamErr *streamerror.Error) error {
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

	if s.getState() == inBinded && streamErr != nil && streamErr.Reason == streamerror.ConnectionTimeout {
		s.discTm = time.AfterFunc(disconnectTimeout, func() {
			s.rq.Run(func() {
				fnCtx, cancel := s.requestContext()
				defer cancel()
				_ = s.close(fnCtx, streamErr)
			})
		})
		s.sendDisabled = true // avoid sending anymore stanzas while closing
		return nil
	}
	return s.close(ctx, streamErr)
}

func (s *inC2S) close(ctx context.Context, disconnectErr error) error {
	switch s.getState() {
	case inDisconnected:
		return s.terminate(ctx) // disconnected... terminate stream
	case inTerminated:
		return nil // terminated... we're done here
	default:
		break
	}
	s.setState(inDisconnected)

	if s.discTm != nil {
		s.discTm.Stop()
	}
	// run disconnected C2S hook
	halted, err := s.runHook(ctx, hook.C2SStreamDisconnected, &hook.C2SStreamInfo{
		ID:              s.ID().String(),
		JID:             s.JID(),
		Presence:        s.Presence(),
		DisconnectError: disconnectErr,
	})
	if halted {
		return nil
	}
	if err != nil {
		return err
	}
	return s.terminate(ctx)
}

func (s *inC2S) terminate(ctx context.Context) error {
	// unregister C2S stream
	if err := s.router.C2S().Unregister(s); err != nil {
		return err
	}
	// delete cluster resource
	if err := s.resMng.DelResource(ctx, s.Username(), s.Resource()); err != nil {
		return err
	}
	reportConnectionUnregistered()

	// close underlying transport
	_ = s.tr.Close()

	_, err := s.runHook(ctx, hook.C2SStreamTerminated, &hook.C2SStreamInfo{
		ID:  s.ID().String(),
		JID: s.JID(),
	})
	if err != nil {
		return err
	}
	close(s.doneCh) // signal termination

	s.setState(inTerminated)
	return nil
}

func (s *inC2S) restartSession() {
	_ = s.session.Reset(s.tr)
	s.setState(inConnecting)
}

func (s *inC2S) sendElement(ctx context.Context, elem stravaganza.Element) error {
	if s.sendDisabled {
		return nil
	}
	_ = s.session.Send(ctx, elem)

	reportOutgoingRequest(
		elem.Name(),
		elem.Attribute(stravaganza.Type),
	)
	// run element sent hook
	_, err := s.runHook(ctx, hook.C2SStreamElementSent, &hook.C2SStreamInfo{
		ID:      s.ID().String(),
		JID:     s.JID(),
		Element: elem,
	})
	return err
}

func (s *inC2S) getResource() c2smodel.ResourceDesc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return c2smodel.NewResourceDesc(
		instance.ID(),
		s.jd,
		s.pr,
		s.inf.ReadOnly(),
	)
}

func (s *inC2S) updateRateLimiter() error {
	j := s.JID()
	rLim := s.shapers.MatchingJID(j).RateLimiter()
	return s.tr.SetReadRateLimiter(rLim)
}

func (s *inC2S) setJID(jd *jid.JID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jd = jd
}

func (s *inC2S) setPresence(pr *stravaganza.Presence) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pr = pr
}

func (s *inC2S) setState(state state) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

func (s *inC2S) getState() state {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *inC2S) runHook(ctx context.Context, hookName string, inf *hook.C2SStreamInfo) (halt bool, err error) {
	return s.hk.Run(hookName, &hook.ExecutionContext{
		Info:    inf,
		Sender:  s,
		Context: ctx,
	})
}

func (s *inC2S) requestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.cfg.reqTimeout)
}

var currentID uint64

func nextStreamID() stream.C2SID {
	return stream.C2SID(atomic.AddUint64(&currentID, 1))
}
