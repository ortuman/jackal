/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2srouter

import (
	"context"
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/s2s"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

type remoteRouter struct {
	remoteDomain string
	localDomain  string
	outProvider  s2s.OutProvider
	mu           sync.RWMutex
	outStm       stream.S2SOut
}

func newRemoteRouter(domain, localDomain string, outProvider s2s.OutProvider) *remoteRouter {
	return &remoteRouter{
		remoteDomain: domain,
		localDomain:  localDomain,
		outProvider:  outProvider,
	}
}

func (r *remoteRouter) route(ctx context.Context, stanza xmpp.Stanza) error {
	r.mu.RLock()
	stm := r.outStm
	r.mu.RUnlock()

	if stm == nil {
		r.mu.Lock()
		stm = r.outStm
		if stm == nil {
			outStm, err := r.outProvider.GetOut(ctx, r.localDomain, r.remoteDomain)
			if err != nil {
				r.mu.Unlock()
				log.Error(err)
				return router.ErrFailedRemoteConnect
			}
			stm = outStm
			r.outStm = stm
		}
		r.mu.Unlock()
	}
	stm.SendElement(ctx, stanza)
	return nil
}
