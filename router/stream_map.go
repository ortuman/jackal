/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"sync"

	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type streamBucket struct {
	mu      sync.RWMutex
	streams map[string][]stream.C2S
}

func (b *streamBucket) bind(stm stream.C2S) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if streams := b.streams[stm.Username()]; streams != nil {
		res := stm.Resource()
		for _, usrStream := range streams {
			if usrStream.Resource() == res {
				return // already bound
			}
		}
		b.streams[stm.Username()] = append(streams, stm)
	} else {
		b.streams[stm.Username()] = []stream.C2S{stm}
	}
}

func (b *streamBucket) unbind(j *jid.JID) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	username := j.Node()

	found := false
	if streams := b.streams[username]; streams != nil {
		res := j.Resource()
		for i := 0; i < len(streams); i++ {
			if res == streams[i].Resource() {
				streams = append(streams[:i], streams[i+1:]...)
				if len(streams) > 0 {
					b.streams[username] = streams
				} else {
					delete(b.streams, username)
				}
				found = true
				break
			}
		}
	}
	return found
}

func (b *streamBucket) route(element xmpp.Stanza, toJID *jid.JID) error {
	recipients := b.streams[toJID.Node()]
	if len(recipients) == 0 {
		exists, err := storage.UserExists(toJID.Node())
		if err != nil {
			return err
		}
		if exists {
			return ErrNotAuthenticated
		}
		return ErrNotExistingAccount
	}
	if toJID.IsFullWithUser() {
		for _, stm := range recipients {
			if stm.Resource() == toJID.Resource() {
				stm.SendElement(element)
				return nil
			}
		}
		return ErrResourceNotFound
	}
	switch element.(type) {
	case *xmpp.Message:
		// send to highest priority stream
		stm := recipients[0]
		var highestPriority int8
		if p := stm.Presence(); p != nil {
			highestPriority = p.Priority()
		}
		for i := 1; i < len(recipients); i++ {
			rcp := recipients[i]
			if p := rcp.Presence(); p != nil && p.Priority() > highestPriority {
				stm = rcp
				highestPriority = p.Priority()
			}
		}
		stm.SendElement(element)

	default:
		// broadcast toJID all streams
		for _, stm := range recipients {
			stm.SendElement(element)
		}
	}
	return nil
}
