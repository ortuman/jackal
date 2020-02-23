/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"crypto/tls"
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router/host"
	"github.com/ortuman/jackal/stream"
)

type OutProvider interface {
	GetOut(localDomain, remoteDomain string) stream.S2SOut

	Shutdown(ctx context.Context) error

	newOut(localDomain, remoteDomain string) *outStream
}

type outProvider struct {
	cfg            *Config
	hosts          *host.Hosts
	dialer         Dialer
	mu             sync.RWMutex
	outConnections map[string]stream.S2SOut
}

func NewOutProvider(config *Config, hosts *host.Hosts) OutProvider {
	return &outProvider{
		cfg:            config,
		hosts:          hosts,
		dialer:         newDialer(),
		outConnections: make(map[string]stream.S2SOut),
	}
}

func (p *outProvider) GetOut(localDomain, remoteDomain string) stream.S2SOut {
	domainPair := getDomainPair(localDomain, remoteDomain)
	p.mu.RLock()
	outStm := p.outConnections[domainPair]
	p.mu.RUnlock()

	if outStm != nil {
		return outStm
	}
	p.mu.Lock()
	outStm = p.outConnections[domainPair] // 2nd check
	if outStm != nil {
		p.mu.Unlock()
		return outStm
	}
	outStm = p.newOut(localDomain, remoteDomain)
	p.outConnections[domainPair] = outStm
	p.mu.Unlock()

	log.Infof("registered s2s out stream... (domainpair: %s)", domainPair)

	return outStm
}

func (p *outProvider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.outConnections) == 0 {
		return nil
	}
	for k, conn := range p.outConnections {
		conn.Disconnect(ctx, nil)
		delete(p.outConnections, k)
	}
	log.Infof("%s: closed %d out connection(s)", len(p.outConnections))

	return nil
}

func (p *outProvider) newOut(localDomain, remoteDomain string) *outStream {
	tlsConfig := &tls.Config{
		MaxVersion:   tls.VersionTLS11,
		ServerName:   remoteDomain,
		Certificates: p.hosts.Certificates(),
	}
	cfg := &outConfig{
		keyGen:        &keyGen{secret: p.cfg.DialbackSecret},
		timeout:       p.cfg.Timeout,
		localDomain:   localDomain,
		remoteDomain:  remoteDomain,
		keepAlive:     p.cfg.Transport.KeepAlive,
		tls:           tlsConfig,
		maxStanzaSize: p.cfg.MaxStanzaSize,
	}
	return newOutStream(cfg, p.hosts, p.dialer)
}

func getDomainPair(localDomain, remoteDomain string) string {
	return localDomain + ":" + remoteDomain
}
