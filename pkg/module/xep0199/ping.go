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

package xep0199

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/log/level"

	kitlog "github.com/go-kit/log"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza"
	stanzaerror "github.com/jackal-xmpp/stravaganza/errors/stanza"
	streamerror "github.com/jackal-xmpp/stravaganza/errors/stream"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
)

const pingNamespace = "urn:xmpp:ping"

const (
	// ModuleName represents ping module name.
	ModuleName = "ping"

	// XEPNumber represents ping XEP number.
	XEPNumber = "0199"
)

const (
	modRequestTimeout = time.Second * 5

	killAction = "kill"
)

// Config contains ping module configuration options.
type Config struct {
	// AckTimeout tells how long should we wait until considering a client to be disconnected.
	AckTimeout time.Duration `fig:"ack_timeout" default:"32s"`
	// Interval tells how often pings should be sent to clients.
	Interval time.Duration `fig:"interval" default:"1m"`
	// SendPings tells whether server pings should be sent.
	SendPings bool `fig:"send_pings"`
	// TimeoutAction specifies the action to be taken when a client is considered as disconnected.
	TimeoutAction string `fig:"timeout_action" default:"none"`
}

// Ping represents ping (XEP-0199) module type.
type Ping struct {
	cfg    Config
	router router.Router
	hk     *hook.Hooks
	logger kitlog.Logger

	mu         sync.RWMutex
	pingTimers map[string]*time.Timer
	ackTimers  map[string]*time.Timer
}

// New returns a new initialized ping instance.
func New(cfg Config, router router.Router, hk *hook.Hooks, logger kitlog.Logger) *Ping {
	return &Ping{
		cfg:        cfg,
		router:     router,
		hk:         hk,
		logger:     kitlog.With(logger, "module", ModuleName, "xep", XEPNumber),
		pingTimers: make(map[string]*time.Timer),
		ackTimers:  make(map[string]*time.Timer),
	}
}

// Name returns ping module name.
func (p *Ping) Name() string { return ModuleName }

// StreamFeature returns ping module stream feature.
func (p *Ping) StreamFeature(_ context.Context, _ string) (stravaganza.Element, error) {
	return nil, nil
}

// ServerFeatures returns ping server disco features.
func (p *Ping) ServerFeatures(_ context.Context) ([]string, error) {
	return []string{pingNamespace}, nil
}

// AccountFeatures returns ping account disco features.
func (p *Ping) AccountFeatures(_ context.Context) ([]string, error) {
	return nil, nil
}

// Start starts ping module.
func (p *Ping) Start(_ context.Context) error {
	if p.cfg.SendPings {
		p.hk.AddHook(hook.C2SStreamBinded, p.onBinded, hook.DefaultPriority)
		p.hk.AddHook(hook.C2SStreamDisconnected, p.onDisconnect, hook.HighestPriority)
		p.hk.AddHook(hook.C2SStreamElementReceived, p.onRecvElement, hook.HighestPriority)
	}
	level.Info(p.logger).Log("msg", "started ping module")
	return nil
}

// Stop stops ping module.
func (p *Ping) Stop(_ context.Context) error {
	if p.cfg.SendPings {
		p.hk.RemoveHook(hook.C2SStreamBinded, p.onBinded)
		p.hk.RemoveHook(hook.C2SStreamDisconnected, p.onDisconnect)
		p.hk.RemoveHook(hook.C2SStreamElementReceived, p.onRecvElement)
	}
	level.Info(p.logger).Log("msg", "stopped ping module")
	return nil
}

// MatchesNamespace tells whether namespace matches ping module.
func (p *Ping) MatchesNamespace(namespace string, _ bool) bool {
	return namespace == pingNamespace
}

// ProcessIQ process a ping iq.
func (p *Ping) ProcessIQ(ctx context.Context, iq *stravaganza.IQ) error {
	switch {
	case isPingIQ(iq):
		return p.sendPongReply(ctx, iq)
	default:
		_, _ = p.router.Route(ctx, xmpputil.MakeErrorStanza(iq, stanzaerror.BadRequest))
		return nil
	}
}

func (p *Ping) sendPongReply(ctx context.Context, pingIQ *stravaganza.IQ) error {
	pongIQ := xmpputil.MakeResultIQ(pingIQ, nil)
	_, _ = p.router.Route(ctx, pongIQ)
	return nil
}

func (p *Ping) onBinded(_ context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	p.schedulePing(inf.JID)
	return nil
}

func (p *Ping) onRecvElement(_ context.Context, execCtx *hook.ExecutionContext) error {
	stm := execCtx.Sender.(stream.C2S)
	if !stm.IsBinded() {
		return nil
	}
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	p.cancelTimers(inf.JID)
	p.schedulePing(inf.JID)
	return nil
}

func (p *Ping) onDisconnect(_ context.Context, execCtx *hook.ExecutionContext) error {
	inf := execCtx.Info.(*hook.C2SStreamInfo)
	if jd := inf.JID; jd != nil {
		p.cancelTimers(jd)
	}
	return nil
}

func (p *Ping) schedulePing(jd *jid.JID) {
	p.mu.Lock()
	p.pingTimers[jd.String()] = time.AfterFunc(p.cfg.Interval, func() {
		p.sendPing(jd)
	})
	p.mu.Unlock()
}

func (p *Ping) sendPing(jd *jid.JID) {
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, uuid.New().String()).
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithAttribute(stravaganza.From, jd.Domain()).
		WithAttribute(stravaganza.To, jd.String()).
		WithChild(
			stravaganza.NewBuilder("ping").
				WithAttribute(stravaganza.Namespace, "urn:xmpp:ping").
				Build(),
		).
		BuildIQ()

	// send ping IQ
	ctx, cancel := context.WithTimeout(context.Background(), modRequestTimeout)
	defer cancel()

	_, _ = p.router.Route(ctx, iq)

	// schedule ack timeout
	p.mu.Lock()
	p.ackTimers[jd.String()] = time.AfterFunc(p.cfg.AckTimeout, func() {
		p.timeout(jd)
	})
	p.mu.Unlock()

	level.Info(p.logger).Log("msg", "sent ping", "jid", jd.String())
}

func (p *Ping) timeout(jd *jid.JID) {
	// perform timeout action
	switch p.cfg.TimeoutAction {
	case killAction:
		if stm := p.router.C2S().LocalStream(jd.Node(), jd.Resource()); stm != nil {
			_ = stm.Disconnect(streamerror.E(streamerror.ConnectionTimeout))
		}
	}
	level.Info(p.logger).Log("msg", "stream timeout", "jid", jd.String())
}

func (p *Ping) cancelTimers(jd *jid.JID) {
	jk := jd.String()
	p.mu.Lock()
	if tm := p.pingTimers[jk]; tm != nil {
		tm.Stop()
	}
	if tm := p.ackTimers[jk]; tm != nil {
		tm.Stop()
	}
	delete(p.pingTimers, jk)
	delete(p.ackTimers, jk)
	p.mu.Unlock()
}

func isPingIQ(iq *stravaganza.IQ) bool {
	return iq.IsGet() && iq.ChildNamespace("ping", pingNamespace) != nil
}
