/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2srouter

import (
	"context"
	"sync"

	"github.com/ortuman/jackal/s2s"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

type remoteRouter struct {
	mu           sync.RWMutex
	localDomain  string
	remoteDomain string
	outProvider  s2s.OutProvider
	outStm       stream.S2SOut
}

func newRemoteRouter(localDomain, remoteDomain string, outProvider s2s.OutProvider) *remoteRouter {
	return &remoteRouter{
		localDomain:  localDomain,
		remoteDomain: remoteDomain,
		outProvider:  outProvider,
	}
}

func (r *remoteRouter) route(ctx context.Context, stanza xmpp.Stanza) {
	r.mu.RLock()
	ready := r.outStm != nil
	r.mu.RUnlock()

	if !ready {
		r.mu.Lock()
		if r.outStm == nil {
			r.outStm = r.outProvider.GetOut(r.localDomain, r.remoteDomain)
		}
		r.mu.Unlock()
	}
	r.outStm.SendElement(ctx, stanza)
}
