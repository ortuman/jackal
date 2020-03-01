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
	"github.com/ortuman/jackal/xmpp"
)

type s2sRouter struct {
	mu          sync.RWMutex
	outProvider s2s.OutProvider
	remotes     map[string]*remoteRouter
}

func New(outProvider s2s.OutProvider) router.S2SRouter {
	return &s2sRouter{
		outProvider: outProvider,
		remotes:     make(map[string]*remoteRouter),
	}
}

func (r *s2sRouter) Route(ctx context.Context, stanza xmpp.Stanza, localDomain string) error {
	remoteDomain := stanza.ToJID().Domain()

	r.mu.RLock()
	rr := r.remotes[remoteDomain]
	r.mu.RUnlock()

	if rr == nil {
		r.mu.Lock()
		rr = r.remotes[remoteDomain] // avoid double initialization
		if rr == nil {
			rr = newRemoteRouter(localDomain, remoteDomain, r.outProvider)
			r.remotes[remoteDomain] = rr
		}
		r.mu.Unlock()
	}
	rr.route(ctx, stanza)

	return nil
}
