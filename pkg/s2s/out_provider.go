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
	"time"

	kitlog "github.com/go-kit/log"

	"github.com/go-kit/log/level"

	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/ortuman/jackal/pkg/shaper"
)

// OutProvider is an outgoing S2S stream provider.
type OutProvider struct {
	cfg     OutConfig
	hosts   *host.Hosts
	kv      kv.KV
	shapers shaper.Shapers
	hk      *hook.Hooks
	logger  kitlog.Logger

	mu         sync.RWMutex
	outStreams map[string]s2sOut
	doneCh     chan chan struct{}

	newOutFn func(sender, target string) s2sOut
	newDbFn  func(sender, target string, dbParam DialbackParams) s2sDialback
}

// NewOutProvider creates and initializes a new OutProvider instance.
func NewOutProvider(
	cfg OutConfig,
	hosts *host.Hosts,
	kv kv.KV,
	shapers shaper.Shapers,
	hk *hook.Hooks,
	logger kitlog.Logger,
) *OutProvider {
	op := &OutProvider{
		cfg:        cfg,
		hosts:      hosts,
		shapers:    shapers,
		kv:         kv,
		hk:         hk,
		logger:     logger,
		outStreams: make(map[string]s2sOut),
		doneCh:     make(chan chan struct{}),
	}
	op.newOutFn = op.newOutS2S
	op.newDbFn = op.newDialbackS2S
	return op
}

// DialbackSecret returns dialback secret value.
func (p *OutProvider) DialbackSecret() string {
	return p.cfg.DialbackSecret
}

// GetOut returns associated outgoing S2S stream given a sender-target pair domain.
func (p *OutProvider) GetOut(ctx context.Context, sender, target string) (stream.S2SOut, error) {
	domainPair := getDomainPair(sender, target)

	p.mu.RLock()
	outStm := p.outStreams[domainPair]
	p.mu.RUnlock()

	if outStm != nil {
		return outStm, nil
	}
	p.mu.Lock()
	outStm = p.outStreams[domainPair] // 2nd check
	if outStm != nil {
		p.mu.Unlock()
		return outStm, nil
	}
	outStm = p.newOutFn(sender, target)
	p.outStreams[domainPair] = outStm
	p.mu.Unlock()

	if err := outStm.dial(ctx); err != nil {
		p.mu.Lock()
		delete(p.outStreams, domainPair)
		p.mu.Unlock()
		level.Warn(p.logger).Log("msg", "failed to dial outgoing S2S stream",
			"err", err, "sender", sender, "target", target,
		)
		return nil, err
	}
	go func() {
		if err := outStm.start(); err != nil {
			p.mu.Lock()
			delete(p.outStreams, domainPair)
			p.mu.Unlock()
			level.Warn(p.logger).Log("msg", "failed to start outgoing S2S stream",
				"err", err, "sender", sender, "target", target,
			)
			return
		}
	}()
	return outStm, nil
}

// GetDialback returns associated dialback S2S stream given a sender-target pair domain and a parameters set.
func (p *OutProvider) GetDialback(ctx context.Context, sender, target string, params DialbackParams) (stream.S2SDialback, error) {
	outStm := p.newDbFn(sender, target, params)
	if err := outStm.dial(ctx); err != nil {
		level.Warn(p.logger).Log("msg", "failed to dial S2S dialback stream",
			"err", err, "sender", sender, "target", target,
		)
		return nil, err
	}
	go func() {
		if err := outStm.start(); err != nil {
			level.Warn(p.logger).Log("msg", "failed to start S2S dialback stream",
				"err", err, "sender", sender, "target", target,
			)
			return
		}
	}()
	return outStm, nil
}

// Start starts S2S out provider.
func (p *OutProvider) Start(_ context.Context) error {
	go p.reportMetrics()
	level.Info(p.logger).Log("msg", "started S2S out provider")
	return nil
}

// Stop stops S2S out provider.
func (p *OutProvider) Stop(ctx context.Context) error {
	// stop metrics reporting
	ch := make(chan struct{})
	p.doneCh <- ch
	<-ch

	var stms []s2sOut

	// grab all connections
	p.mu.RLock()
	for _, stm := range p.outStreams {
		stms = append(stms, stm)
	}
	p.mu.RUnlock()

	// perform stream disconnection
	var wg sync.WaitGroup
	for _, s := range stms {
		wg.Add(1)
		go func(stm s2sOut) {
			defer wg.Done()
			select {
			case <-stm.Disconnect(streamerror.E(streamerror.SystemShutdown)):
				break
			case <-ctx.Done():
				break
			}
		}(s)
	}
	wg.Wait()

	level.Info(p.logger).Log("msg", "stopped S2S out provider", "total_connections", len(stms))
	return nil
}

func (p *OutProvider) unregister(stm *outS2S) {
	id := stm.ID()
	domainPair := getDomainPair(id.Sender, id.Target)
	p.mu.Lock()
	delete(p.outStreams, domainPair)
	p.mu.Unlock()
}

func (p *OutProvider) newOutS2S(sender, target string) s2sOut {
	return newOutS2S(
		sender,
		target,
		p.tlsConfig(target),
		p.hosts,
		p.kv,
		p.shapers,
		p.hk,
		p.logger,
		p.unregister,
		outConfig{
			dbSecret:         p.cfg.DialbackSecret,
			dialTimeout:      p.cfg.DialTimeout,
			keepAliveTimeout: p.cfg.KeepAliveTimeout,
			reqTimeout:       p.cfg.RequestTimeout,
			maxStanzaSize:    p.cfg.MaxStanzaSize,
		},
	)
}

func (p *OutProvider) newDialbackS2S(sender, target string, dbParams DialbackParams) s2sDialback {
	return newDialbackS2S(
		sender,
		target,
		p.tlsConfig(target),
		p.hosts,
		p.shapers,
		p.logger,
		outConfig{
			dbSecret:         p.cfg.DialbackSecret,
			dialTimeout:      p.cfg.DialTimeout,
			keepAliveTimeout: p.cfg.KeepAliveTimeout,
			reqTimeout:       p.cfg.RequestTimeout,
			maxStanzaSize:    p.cfg.MaxStanzaSize,
		},
		dbParams,
	)
}

func (p *OutProvider) tlsConfig(serverName string) *tls.Config {
	return &tls.Config{
		ServerName:   serverName,
		Certificates: p.hosts.Certificates(),
	}
}

func (p *OutProvider) reportMetrics() {
	tc := time.NewTicker(reportTotalConnectionsInterval)
	defer tc.Stop()

	for {
		select {
		case <-tc.C:
			p.mu.RLock()
			totalConns := len(p.outStreams)
			p.mu.RUnlock()
			reportTotalOutgoingConnections(totalConns)

		case ch := <-p.doneCh:
			close(ch)
			return
		}
	}
}

func getDomainPair(sender, target string) string {
	return sender + ":" + target
}
