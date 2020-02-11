/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"crypto/tls"
	"sync"
	"sync/atomic"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router/host"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/xmpp"
)

type OutProvider interface {
	GetOut(ctx context.Context, localDomain, remoteDomain string) (stream.S2SOut, error)

	Shutdown(ctx context.Context) error

	getVerifyOut(ctx context.Context, localDomain, remoteDomain string, verifyElem xmpp.XElement) (*outStream, error)
}

type outProvider struct {
	cfg            *Config
	hosts          *host.Hosts
	dialer         Dialer
	mu             sync.RWMutex
	outConnections map[string]stream.S2SOut
	isShuttingDown int32
}

func NewOutProvider(config *Config, hosts *host.Hosts) OutProvider {
	return &outProvider{
		cfg:            config,
		hosts:          hosts,
		dialer:         newDialer(),
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

	if err := p.startOut(ctx, outStm.(*outStream), localDomain, remoteDomain, nil, p.unregisterOut); err != nil {
		p.mu.Lock()
		delete(p.outConnections, domainPair) // something went wrong... wipe out connection
		p.mu.Unlock()
		return nil, err
	}
	log.Infof("registered s2s out stream... (domainpair: %s)", domainPair)

	return outStm, nil
}

func (p *outProvider) getVerifyOut(ctx context.Context, localDomain, remoteDomain string, verifyElem xmpp.XElement) (*outStream, error) {
	outStm := newOutStream(p.hosts)
	if err := p.startOut(ctx, outStm, localDomain, remoteDomain, verifyElem, nil); err != nil {
		return nil, err
	}
	return outStm, nil
}

func (p *outProvider) Shutdown(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.isShuttingDown, 0, 1) {
		return nil // already done
	}
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

func (p *outProvider) startOut(ctx context.Context, outStm *outStream, localDomain, remoteDomain string, verifyElem xmpp.XElement, onDisconnect func(s stream.S2SOut)) error {
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
		dbVerify:        verifyElem,
		onOutDisconnect: onDisconnect,
	}
	if err := outStm.start(ctx, outStreamCfg); err != nil {
		return err
	}
	return nil
}

func (p *outProvider) unregisterOut(stm stream.S2SOut) {
	if atomic.LoadInt32(&p.isShuttingDown) == 1 {
		return // do not unregister stream while shutting down to avoid deadlock
	}
	domainPair := stm.ID()
	p.mu.Lock()
	delete(p.outConnections, domainPair)
	p.mu.Unlock()

	log.Infof("unregistered s2s out stream... (domainpair: %s)", domainPair)
}

func getDomainPair(localDomain, remoteDomain string) string {
	return localDomain + ":" + remoteDomain
}
