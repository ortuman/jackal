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
	"github.com/ortuman/jackal/xmpp"
)

type OutProvider interface {
	GetOut(ctx context.Context, localDomain, remoteDomain string) (stream.S2SOut, error)

	getVerifyOut(dbVerify xmpp.XElement, localDomain, remoteDomain string) (*outStream, error)
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
	domainPair := getDomainPair(localDomain, remoteDomain)
	p.mu.RLock()
	outStm := p.outConnections[domainPair]
	p.mu.RUnlock()

	if outStm != nil {
		return outStm, nil
	}
	p.mu.Lock()
	outStm = p.outConnections[domainPair] // 2nd check
	if outStm != nil {
		p.mu.Unlock()
		return outStm, nil
	}
	outStm = newOutStream(p.hosts)
	p.outConnections[domainPair] = outStm
	p.mu.Unlock()

	if err := p.registerOutStream(ctx, outStm.(*outStream), localDomain, remoteDomain); err != nil {
		p.mu.Lock()
		delete(p.outConnections, domainPair) // something went wrong... wipe out connection
		p.mu.Unlock()
		return nil, err
	}
	return outStm, nil
}

func (p *outProvider) getVerifyOut(dbVerify xmpp.XElement, localDomain, remoteDomain string) (*outStream, error) {
	return nil, nil
}

func (p *outProvider) registerOutStream(ctx context.Context, outStm *outStream, localDomain, remoteDomain string) error {
	conn, err := p.dialer.Dial(ctx, remoteDomain)
	if err != nil {
		return err
	}
	tlsConfig := &tls.Config{
		ServerName:   remoteDomain,
		Certificates: p.hosts.Certificates(),
	}
	outStreamCfg := &outStreamConfig{
		keyGen:          &keyGen{secret: p.cfg.DialbackSecret},
		timeout:         p.cfg.Timeout,
		localDomain:     localDomain,
		remoteDomain:    remoteDomain,
		transport:       transport.NewSocketTransport(conn, p.cfg.Transport.KeepAlive),
		tls:             tlsConfig,
		maxStanzaSize:   p.cfg.MaxStanzaSize,
		onOutDisconnect: p.unregisterOutStream,
	}
	if err := outStm.start(ctx, outStreamCfg); err != nil {
		return err
	}
	log.Infof("registered s2s out stream... (domainpair: %s)", getDomainPair(localDomain, remoteDomain))
	return nil
}

func (p *outProvider) unregisterOutStream(stm stream.S2SOut) {
	domainPair := stm.ID()
	p.mu.Lock()
	delete(p.outConnections, domainPair)
	p.mu.Unlock()

	log.Infof("unregistered s2s out stream... (domainpair: %s)", domainPair)
}

func getDomainPair(localDomain, remoteDomain string) string {
	return localDomain + ":" + remoteDomain
}
