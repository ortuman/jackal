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
	"fmt"
	"sync"
	"time"

	"github.com/jackal-xmpp/sonar"
	streamerror "github.com/jackal-xmpp/stravaganza/v2/errors/stream"
	"github.com/ortuman/jackal/pkg/cluster/kv"
	"github.com/ortuman/jackal/pkg/host"
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/ortuman/jackal/pkg/shaper"
)

// OutProvider is an outgoing S2S stream provider.
type OutProvider struct {
	hosts   *host.Hosts
	cfg     Config
	kv      kv.KV
	shapers shaper.Shapers
	sn      *sonar.Sonar

	mu         sync.RWMutex
	outStreams map[string]s2sOut
	doneCh     chan chan struct{}

	newOutFn func(sender, target string) s2sOut
	newDbFn  func(sender, target string, dbParam DialbackParams) s2sDialback
}

// NewOutProvider creates and initializes a new OutProvider instance.
func NewOutProvider(
	hosts *host.Hosts,
	kv kv.KV,
	shapers shaper.Shapers,
	sn *sonar.Sonar,
	cfg Config,
) *OutProvider {
	op := &OutProvider{
		hosts:      hosts,
		shapers:    shapers,
		kv:         kv,
		sn:         sn,
		cfg:        cfg,
		outStreams: make(map[string]s2sOut),
		doneCh:     make(chan chan struct{}),
	}
	op.newOutFn = op.newOutS2S
	op.newDbFn = op.newDialbackS2S
	return op
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
		log.Warnw(fmt.Sprintf("Failed to dial outgoing S2S stream: %v", err),
			"sender", sender, "target", target,
		)
		return nil, err
	}
	go func() {
		if err := outStm.start(); err != nil {
			p.mu.Lock()
			delete(p.outStreams, domainPair)
			p.mu.Unlock()
			log.Warnw(fmt.Sprintf("Failed to start outgoing S2S stream: %v", err),
				"sender", sender, "target", target,
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
		log.Warnw(fmt.Sprintf("Failed to dial S2S dialback stream: %v", err),
			"sender", sender, "target", target,
		)
		return nil, err
	}
	go func() {
		if err := outStm.start(); err != nil {
			log.Warnw(fmt.Sprintf("Failed to start S2S dialback stream: %v", err),
				"sender", sender, "target", target,
			)
			return
		}
	}()
	return outStm, nil
}

// Start starts S2S out provider.
func (p *OutProvider) Start(_ context.Context) error {
	go p.reportMetrics()
	log.Infow("Started S2S out provider")
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

	log.Infow("Stopped S2S out provider", "total_connections", len(stms))
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
		p.cfg,
		p.kv,
		p.shapers,
		p.sn,
		p.unregister,
	)
}

func (p *OutProvider) newDialbackS2S(sender, target string, dbParam DialbackParams) s2sDialback {
	return newDialbackS2S(
		sender,
		target,
		p.tlsConfig(target),
		p.hosts,
		p.cfg,
		dbParam,
		p.shapers,
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
