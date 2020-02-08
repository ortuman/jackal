/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"context"
	"sync"

	"github.com/ortuman/jackal/stream"
)

type OutProvider interface {
	GetOut(ctx context.Context, localDomain, remoteDomain string) (stream.S2SOut, error)
}

type outProvider struct {
	dialer         Dialer
	mu             sync.RWMutex
	outConnections map[string]stream.S2SOut
}

func NewOutProvider() OutProvider {
	return &outProvider{
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
	// TODO(ortuman) integrate dialer

	return nil, nil
}
