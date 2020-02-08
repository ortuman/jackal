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
	"github.com/ortuman/jackal/transport"
)

type OutProvider interface {
	GetOut(ctx context.Context, localDomain, remoteDomain string) (stream.S2SOut, error)
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
		outConnections: make(map[string]stream.S2SOut),
	}
}

func (p *outProvider) GetOut(ctx context.Context, localDomain, remoteDomain string) (stream.S2SOut, error) {
	domainPair := localDomain + ":" + remoteDomain
	p.mu.RLock()
	outStm := p.outConnections[domainPair]
	p.mu.RUnlock()

	if outStm != nil {
		return outStm, nil
	}
	conn, err := p.dialer.Dial(ctx, remoteDomain)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		ServerName:   remoteDomain,
		Certificates: p.hosts.Certificates(),
	}
	outStreamCfg := &streamConfig{
		keyGen:          &keyGen{secret: p.cfg.DialbackSecret},
		timeout:         p.cfg.Timeout,
		localDomain:     localDomain,
		remoteDomain:    remoteDomain,
		transport:       transport.NewSocketTransport(conn, p.cfg.Transport.KeepAlive),
		tls:             tlsConfig,
		maxStanzaSize:   p.cfg.MaxStanzaSize,
		onOutDisconnect: p.unregisterOutStream,
	}
	log.Infof("registered s2s out stream... (domainpair: %s)", domainPair)

	println(outStreamCfg)

	return nil, nil
}

func (p *outProvider) unregisterOutStream(stm stream.S2SOut) {
	domainPair := stm.ID()
	p.mu.Lock()
	delete(p.outConnections, domainPair)
	p.mu.Unlock()

	log.Infof("unregistered s2s out stream... (domainpair: %s)", domainPair)
}
