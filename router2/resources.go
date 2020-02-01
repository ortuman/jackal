/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router2

import (
	"context"
	"sync"

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
			if stm.Resource() == toJID.Resource() {
				stm.SendElement(ctx, stanza)
				return nil
			}
		}
		return ErrResourceNotFound
	}
	switch stanza.(type) {
	case *xmpp.Message:
		// send to highest priority stream
		stm := r.streams[0]
		var highestPriority int8
		if p := stm.Presence(); p != nil {
			highestPriority = p.Priority()
		}
		for i := 1; i < len(r.streams); i++ {
			rcp := r.streams[i]
			if p := rcp.Presence(); p != nil && p.Priority() > highestPriority {
				stm = rcp
				highestPriority = p.Priority()
			}
		}
		stm.SendElement(ctx, stanza)

	default:
		// broadcast toJID all streams
		for _, stm := range r.streams {
			stm.SendElement(ctx, stanza)
		}
	}
	return nil
}
