/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2srouter

import (
	"context"
	"sync"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
)

type resources struct {
	mu      sync.RWMutex
	streams []stream.C2S
}

func (r *resources) len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.streams)
}

func (r *resources) allStreams() []stream.C2S {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.streams
}

func (r *resources) stream(resource string) stream.C2S {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, stm := range r.streams {
		if stm.Resource() == resource {
			return stm
		}
	}
	return nil
}

func (r *resources) bind(stm stream.C2S) {
	r.mu.Lock()
	defer r.mu.Unlock()

	res := stm.Resource()
	for _, s := range r.streams {
		if s.Resource() == res {
			return
		}
	}
	r.streams = append(r.streams, stm)
}

func (r *resources) unbind(res string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, s := range r.streams {
		if s.Resource() != res {
			continue
		}
		r.streams = append(r.streams[:i], r.streams[i+1:]...)
		return
	}
}

func (r *resources) route(ctx context.Context, stanza xmpp.Stanza) error {
	toJID := stanza.ToJID()
	if toJID.IsFullWithUser() {
		for _, stm := range r.streams {
			if p := stm.Presence(); p != nil && p.IsAvailable() && stm.Resource() == toJID.Resource() {
				stm.SendElement(ctx, stanza)
				return nil
			}
		}
		return router.ErrResourceNotFound
	}
	switch stanza.(type) {
	case *xmpp.Message:
		// send to highest priority stream
		var highestPriority int8
		var recipient stream.C2S

		for _, stm := range r.streams {
			if p := stm.Presence(); p != nil && p.IsAvailable() && p.Priority() > highestPriority {
				recipient = stm
				highestPriority = p.Priority()
			}
		}
		if recipient == nil {
			goto broadcast
		}
		recipient.SendElement(ctx, stanza)
		return nil
	}

broadcast:
	// broadcast toJID all streams
	for _, stm := range r.streams {
		if p := stm.Presence(); p != nil && p.IsAvailable() {
			stm.SendElement(ctx, stanza)
		}
	}
	return nil
}
