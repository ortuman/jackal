/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2srouter

import (
	"context"
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type c2sRouter struct {
	mu           sync.RWMutex
	tbl          map[string]*resources
	userRep      repository.User
	blockListRep repository.BlockList
}

func New(userRep repository.User, blockListRep repository.BlockList) router.C2SRouter {
	return &c2sRouter{
		tbl:          make(map[string]*resources),
		userRep:      userRep,
		blockListRep: blockListRep,
	}
}

func (r *c2sRouter) Route(ctx context.Context, stanza xmpp.Stanza, validateStanza bool) error {
	fromJID := stanza.FromJID()
	toJID := stanza.ToJID()

	// validate if sender JID is blocked
	if validateStanza && r.isBlockedJID(ctx, toJID, fromJID.Node()) {
		return router.ErrBlockedJID
	}
	username := stanza.ToJID().Node()
	r.mu.RLock()
	rs := r.tbl[username]
	r.mu.RUnlock()

	if rs == nil {
		exists, err := r.userRep.UserExists(ctx, username)
		if err != nil {
			return err
		}
		if exists {
			return router.ErrNotAuthenticated
		}
		return router.ErrNotExistingAccount
	}
	return rs.route(ctx, stanza)
}

func (r *c2sRouter) Bind(stm stream.C2S) {
	user := stm.Username()
	r.mu.RLock()
	rs := r.tbl[user]
	r.mu.RUnlock()

	if rs == nil {
		r.mu.Lock()
		rs = r.tbl[user] // avoid double initialization
		if rs == nil {
			rs = &resources{}
			r.tbl[user] = rs
		}
		r.mu.Unlock()
	}
	rs.bind(stm)

	log.Infof("bound c2s stream... (%s/%s)", stm.Username(), stm.Resource())
}

func (r *c2sRouter) Unbind(user, resource string) {
	r.mu.RLock()
	rs := r.tbl[user]
	r.mu.RUnlock()

	if rs == nil {
		return
	}
	r.mu.Lock()
	rs.unbind(resource)
	if rs.len() == 0 {
		delete(r.tbl, user)
	}
	r.mu.Unlock()

	log.Infof("unbound c2s stream... (%s/%s)", user, resource)
}

func (r *c2sRouter) Stream(username, resource string) stream.C2S {
	r.mu.RLock()
	rs := r.tbl[username]
	r.mu.RUnlock()

	if rs == nil {
		return nil
	}
	return rs.stream(resource)
}

func (r *c2sRouter) Streams(username string) []stream.C2S {
	r.mu.RLock()
	rs := r.tbl[username]
	r.mu.RUnlock()

	if rs == nil {
		return nil
	}
	return rs.allStreams()
}

func (r *c2sRouter) isBlockedJID(ctx context.Context, j *jid.JID, username string) bool {
	blockList, err := r.blockListRep.FetchBlockListItems(ctx, username)
	if err != nil {
		log.Error(err)
		return false
	}
	if len(blockList) == 0 {
		return false
	}
	blockListJIDs := make([]jid.JID, len(blockList))
	for i, listItem := range blockList {
		j, _ := jid.NewWithString(listItem.JID, true)
		blockListJIDs[i] = *j
	}
	for _, blockedJID := range blockListJIDs {
		if blockedJID.Matches(j) {
			return true
		}
	}
	return false
}
