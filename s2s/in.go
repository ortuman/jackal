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

	"github.com/jackal-xmpp/runqueue"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/cluster/kv"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/event"
	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module"
	"github.com/ortuman/jackal/module/eventhandler/offline"
	xmppparser "github.com/ortuman/jackal/parser"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/router/stream"
	xmppsession "github.com/ortuman/jackal/session"
	"github.com/ortuman/jackal/shaper"
	"github.com/ortuman/jackal/transport"
)

type inS2SState uint32

const (
	inConnecting inS2SState = iota
	inConnected
	inAuthorizingDialbackKey
	inDisconnected
)

type inS2S struct {
	id          stream.S2SInID
	opts        Options
	tr          transport.Transport
	session     session
	hosts       hosts
	router      router.Router
	comps       components
	mods        modules
	outProvider outProvider
	inHub       *InHub
	kv          kv.KV
	shapers     shaper.Shapers
	sn          *sonar.Sonar
	rq          *runqueue.RunQueue

	mu     sync.RWMutex
	state  uint32
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
	sonar *sonar.Sonar,
	opts Options,
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
		xmppsession.Options{
			MaxStanzaSize: opts.MaxStanzaSize,
		},
	)
	// init stream
	stm := &inS2S{
		id:          id,
		opts:        opts,
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
		sn:          sonar,
		rq:          runqueue.New(id.String(), log.Errorf),
		state:       uint32(inConnecting),
	}
	if opts.UseTLS {
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

func (s *inS2S) start() error {
	s.inHub.register(s)

	log.Infow("Registered S2S incoming stream", "id", s.id)

	// post registered incoming S2S event
	ctx, cancel := s.requestContext()
	err := s.postStreamEvent(ctx, event.S2SInStreamRegistered, &event.S2SStreamEventInfo{
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

	tm := time.AfterFunc(s.opts.ConnectTimeout, s.connTimeout) // schedule connect timeout
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
		tm := time.AfterFunc(s.opts.KeepAlive, s.connTimeout) // schedule read timeout
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
		case sErr != nil:
			s.handleSessionError(ctx, sErr)
		case sErr == nil && elem != nil:
			err := s.handleElement(ctx, elem)
			if err != nil {
				log.Warnw("Failed to process incoming S2S session element", "error", err, "id", s.id)
				_ = s.close(ctx)
				return
			}
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

	sb := stravaganza.NewBuilder("stream:features")
	sb.WithAttribute("xmlns:stream", streamNamespace)
	sb.WithAttribute("version", "1.0")

	if !s.flags.isSecured() {
		sb.WithChild(stravaganza.NewBuilder("starttls").
			WithAttribute(stravaganza.Namespace, tlsNamespace).
			WithChild(
				stravaganza.NewBuilder("required").
					Build(),
			).
			Build(),
		)
		s.setState(inConnected)
		return s.session.OpenStream(ctx, sb.Build())
	}
	if !s.flags.isAuthenticated() {
		sb.WithChild(stravaganza.NewBuilder("mechanisms").
			WithAttribute(stravaganza.Namespace, saslNamespace).
			WithChild(
				stravaganza.NewBuilder("mechanism").
					WithText("EXTERNAL").
					Build(),
			).
			Build(),
		)
	}
	sb.WithChild(stravaganza.NewBuilder("dialback").
		WithAttribute(stravaganza.Namespace, dialbackNamespace).
		Build(),
	)
	s.setState(inConnected)
	return s.session.OpenStream(ctx, sb.Build())
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
			switch stanza := elem.(type) {
			case stravaganza.Stanza:
				return s.processStanza(ctx, stanza)

			default:
				log.Infof("CASE-2")
				return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
			}
		}
		return nil
	}
}

func (s *inS2S) processStanza(ctx context.Context, stanza stravaganza.Stanza) error {
	// post stanza received event
	err := s.postStreamEvent(ctx, event.S2SInStreamStanzaReceived, &event.S2SStreamEventInfo{
		ID:     s.ID().String(),
		Sender: s.sender,
		Target: s.target,
		Stanza: stanza,
	})
	if err != nil {
		return err
	}
	toJID := stanza.ToJID()
	if s.comps.IsComponentHost(toJID.Domain()) {
		return s.comps.ProcessStanza(ctx, stanza)
	}
	// handle stanza
	switch stanza := stanza.(type) {
	case *stravaganza.IQ:
		return s.processIQ(ctx, stanza)
	case *stravaganza.Presence:
		return s.processPresence(ctx, stanza)
	case *stravaganza.Message:
		return s.processMessage(ctx, stanza)
	default:
		log.Infof("CASE-1")
		return s.disconnect(ctx, streamerror.E(streamerror.UnsupportedStanzaType))
	}
}

func (s *inS2S) processIQ(ctx context.Context, iq *stravaganza.IQ) error {
	// post IQ received event
	err := s.postStreamEvent(ctx, event.S2SInStreamIQReceived, &event.S2SStreamEventInfo{
		ID:     s.ID().String(),
		Sender: s.sender,
		Target: s.target,
		Stanza: iq,
	})
	if err != nil {
		return err
	}
	if s.mods.IsModuleIQ(iq) {
		return s.mods.ProcessIQ(ctx, iq)
	}
	err = s.router.Route(ctx, iq)
	switch err {
	case router.ErrResourceNotFound:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.ServiceUnavailable, iq).Element())
	case router.ErrRemoteServerNotFound:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerNotFound, iq).Element())
	case router.ErrRemoteServerTimeout:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerTimeout, iq).Element())
	case router.ErrBlockedSender:
		// sender is a blocked JID
		if iq.IsGet() || iq.IsSet() {
			return s.sendElement(ctx, stanzaerror.E(stanzaerror.ServiceUnavailable, iq).Element())
		}
	}
	return nil
}

func (s *inS2S) processMessage(ctx context.Context, message *stravaganza.Message) error {
	// post message received event
	err := s.postStreamEvent(ctx, event.S2SInStreamMessageReceived, &event.S2SStreamEventInfo{
		ID:     s.ID().String(),
		Sender: s.sender,
		Target: s.target,
		Stanza: message,
	})
	if err != nil {
		return err
	}
	msg := message

sndMessage:
	err = s.router.Route(ctx, msg)
	switch err {
	case router.ErrResourceNotFound:
		// treat the stanza as if it were addressed to <node@domain>
		msg, _ = stravaganza.NewBuilderFromElement(msg).
			WithAttribute(stravaganza.From, message.FromJID().String()).
			WithAttribute(stravaganza.To, message.ToJID().ToBareJID().String()).
			BuildMessage(false)
		goto sndMessage

	case router.ErrNotExistingAccount, router.ErrBlockedSender:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.ServiceUnavailable, message).Element())

	case router.ErrRemoteServerNotFound:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerNotFound, message).Element())

	case router.ErrRemoteServerTimeout:
		return s.sendElement(ctx, stanzaerror.E(stanzaerror.RemoteServerTimeout, message).Element())

	case router.ErrUserNotAvailable:
		if !s.mods.IsEnabled(offline.ModuleName) {
			return s.sendElement(ctx, stanzaerror.E(stanzaerror.ServiceUnavailable, message).Element())
		}
		return s.postStreamEvent(ctx, event.S2SStreamMessageUnrouted, &event.S2SStreamEventInfo{
			ID:     s.ID().String(),
			Sender: s.sender,
			Target: s.target,
			Stanza: message,
		})
	}
	return err
}

func (s *inS2S) processPresence(ctx context.Context, presence *stravaganza.Presence) error {
	// post presence received event
	err := s.postStreamEvent(ctx, event.S2SInStreamPresenceReceived, &event.S2SStreamEventInfo{
		ID:     s.ID().String(),
		Sender: s.sender,
		Target: s.target,
		Stanza: presence,
	})
	if err != nil {
		return err
	}
	if presence.ToJID().IsFullWithUser() {
		_ = s.router.Route(ctx, presence)
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
	log.Infow("Failed S2S incoming stream authentication",
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
	log.Infow("Authenticated S2S incoming stream",
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
		dbRes := <-dbOut.Done()
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
		log.Infow("Authorized S2S dialback key",
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
		s.opts.DialbackSecret,
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
		log.Infow("Failed to verify S2S dialback key",
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

	log.Infow("Secured S2S incoming stream", "id", s.id, "sender", s.sender, "target", s.target)

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

func (s *inS2S) close(ctx context.Context) error {
	// unregister S2S stream
	s.inHub.unregister(s)
	s.setState(inDisconnected)

	log.Infow("Unregistered S2S incoming stream",
		"id", s.id,
		"sender", s.sender,
		"target", s.target,
	)
	// post unregistered incoming S2S event
	err := s.postStreamEvent(ctx, event.S2SInStreamUnregistered, &event.S2SStreamEventInfo{
		ID: s.ID().String(),
	})
	if err != nil {
		return err
	}
	reportIncomingConnectionUnregistered()

	// close underlying transport
	return s.tr.Close()
}

func (s *inS2S) sendElement(ctx context.Context, elem stravaganza.Element) error {
	var err error
	err = s.session.Send(ctx, elem)
	reportOutgoingRequest(
		elem.Name(),
		elem.Attribute(stravaganza.Type),
	)
	return err
}

func (s *inS2S) setState(state inS2SState) {
	atomic.StoreUint32(&s.state, uint32(state))
}

func (s *inS2S) getState() inS2SState {
	return inS2SState(atomic.LoadUint32(&s.state))
}

func (s *inS2S) postStreamEvent(ctx context.Context, eventName string, inf *event.S2SStreamEventInfo) error {
	return s.sn.Post(ctx, sonar.NewEventBuilder(eventName).
		WithInfo(inf).
		WithSender(s).
		Build(),
	)
}

func (s *inS2S) requestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.opts.RequestTimeout)
}

var currentID uint64

func nextStreamID() stream.S2SInID {
	return stream.S2SInID(atomic.AddUint64(&currentID, 1))
}
