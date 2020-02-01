/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router2

import (
	"context"
	"sync"

	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

type localRouter struct {
	mu      sync.RWMutex
	tbl     map[string]*resources
	userRep repository.User
}

func newLocalRouter(userRep repository.User) *localRouter {
	return &localRouter{
		tbl:     make(map[string]*resources),
		userRep: userRep,
	}
}

func (r *localRouter) Route(ctx context.Context, stanza xmpp.Stanza) error {
	username := stanza.ToJID().Node()
	r.mu.RLock()
	resources := r.tbl[username]
	r.mu.RUnlock()

	if resources == nil {
		exists, err := r.userRep.UserExists(ctx, username)
		if err != nil {
			return err
		}
		if exists {
			return ErrNotAuthenticated
		}
		return ErrNotExistingAccount
	}
	return resources.route(ctx, stanza)
}

func (r *localRouter) bind(stm stream.C2S) {
	user := stm.Username()

	r.mu.RLock()
	res := r.tbl[user]
	r.mu.RUnlock()

	if res == nil {
		r.mu.Lock()
		res = r.tbl[user] // double check
		if res == nil {
			res = &resources{}
			r.tbl[user] = res
		}
		r.mu.Unlock()
	}
	res.bind(stm)
}

func (r *localRouter) unbind(user, resource string) {
	r.mu.RLock()
	res := r.tbl[user]
	r.mu.RUnlock()

	if res == nil {
		return
	}
	r.mu.Lock()
	res.unbind(resource)
	if res.len() == 0 {
		delete(r.tbl, user)
	}
	r.mu.Unlock()
}
