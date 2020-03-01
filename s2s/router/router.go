/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2srouter

import (
	"context"
	"sync"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/s2s"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

type s2sRouter struct {
	mu          sync.RWMutex
	outProvider s2s.OutProvider
	remotes     map[string]stream.S2SOut
}

func New(outProvider s2s.OutProvider) router.S2SRouter {
	return &s2sRouter{
		outProvider: outProvider,
		remotes:     make(map[string]stream.S2SOut),
	}
}

func (r *s2sRouter) Route(ctx context.Context, stanza xmpp.Stanza, localDomain string) error {
	domain := stanza.ToJID().Domain()

	r.mu.RLock()
	outStm := r.remotes[domain]
	r.mu.RUnlock()

	if outStm == nil {
		r.mu.Lock()
		outStm = r.remotes[domain] // avoid double initialization
		if outStm == nil {
			outStm = r.outProvider.GetOut(localDomain, domain)
			r.remotes[domain] = outStm
		}
		r.mu.Unlock()
	}
	outStm.SendElement(ctx, stanza)
	return nil
}
