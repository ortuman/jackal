// Copyright 2021 The jackal Authors
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

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
	stanzaerror "github.com/jackal-xmpp/stravaganza/v2/errors/stanza"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/module"
	"github.com/ortuman/jackal/pkg/module/hook"
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
	AckTimeout time.Duration

	// Interval tells how often pings should be sent to clients.
	Interval time.Duration

	// SendPings tells whether or not server pings should be sent.
	SendPings bool

	// TimeoutAction specifies the action to be taken when a client is considered as disconnected.
	TimeoutAction string
}

// Ping represents ping (XEP-0199) module type.
type Ping struct {
	cfg    Config
	router router.Router
	mh     *module.Hooks

	mu         sync.RWMutex
	pingTimers map[string]*time.Timer
	ackTimers  map[string]*time.Timer
}

// New returns a new initialized ping instance.
func New(router router.Router, mh *module.Hooks, cfg Config) *Ping {
	return &Ping{
		cfg:        cfg,
		router:     router,
		mh:         mh,
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
		p.mh.AddHook(hook.C2SStreamBinded, p.onBinded, module.DefaultPriority)
		p.mh.AddHook(hook.C2SStreamUnregistered, p.onUnregister, module.DefaultPriority)
		p.mh.AddHook(hook.C2SStreamElementReceived, p.onRecvElement, module.DefaultPriority)
	}
	log.Infow("Started ping module", "xep", XEPNumber)
	return nil
}

// Stop stops ping module.
func (p *Ping) Stop(_ context.Context) error {
	if p.cfg.SendPings {
		p.mh.RemoveHook(hook.C2SStreamBinded, p.onBinded)
		p.mh.RemoveHook(hook.C2SStreamUnregistered, p.onUnregister)
		p.mh.RemoveHook(hook.C2SStreamElementReceived, p.onRecvElement)
	}
	log.Infow("Stopped ping module", "xep", XEPNumber)
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

func (p *Ping) onBinded(_ context.Context, execCtx *module.HookExecutionContext) (halt bool, err error) {
	inf := execCtx.Info.(*hook.C2SStreamHookInfo)
	p.schedulePing(inf.JID)
	return false, nil
}

func (p *Ping) onRecvElement(_ context.Context, execCtx *module.HookExecutionContext) (halt bool, err error) {
	stm := execCtx.Sender.(stream.C2S)
	if !stm.IsBinded() {
		return false, nil
	}
	inf := execCtx.Info.(*hook.C2SStreamHookInfo)
	p.cancelTimers(inf.JID)
	p.schedulePing(inf.JID)
	return false, nil
}

func (p *Ping) onUnregister(_ context.Context, execCtx *module.HookExecutionContext) (halt bool, err error) {
	inf := execCtx.Info.(*hook.C2SStreamHookInfo)
	if jd := inf.JID; jd != nil {
		p.cancelTimers(jd)
	}
	return false, nil
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

	log.Infow("Sent ping", "jid", jd.String(), "xep", XEPNumber)
}

func (p *Ping) timeout(jd *jid.JID) {
	// perform timeout action
	switch p.cfg.TimeoutAction {
	case killAction:
		if stm := p.router.C2S().LocalStream(jd.Node(), jd.Resource()); stm != nil {
			_ = stm.Disconnect(streamerror.E(streamerror.ConnectionTimeout))
		}
	}
	log.Infow("Stream timeout", "jid", jd.String(), "xep", XEPNumber)
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
