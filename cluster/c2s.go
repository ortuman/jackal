/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"sync"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type C2S struct {
	identifier string
	cluster    *Cluster
	node       string
	jid        *jid.JID
	presenceMu sync.RWMutex
	presence   *xmpp.Presence
}

func newC2S(identifier string, jid *jid.JID, presence *xmpp.Presence, node string, cluster *Cluster) *C2S {
	s := &C2S{
		identifier: identifier,
		cluster:    cluster,
		node:       node,
		jid:        jid,
		presence:   presence,
	}
	return s
}

func (s *C2S) ID() string {
	return s.identifier
}

func (s *C2S) Context() *stream.Context {
	return nil
}

func (s *C2S) Username() string {
	return s.jid.Node()
}

func (s *C2S) Domain() string {
	return s.jid.Domain()
}

func (s *C2S) Resource() string {
	return s.jid.Resource()
}

func (s *C2S) JID() *jid.JID {
	return s.jid
}

func (s *C2S) IsSecured() bool       { return true }
func (s *C2S) IsAuthenticated() bool { return true }

func (s *C2S) SetPresence(presence *xmpp.Presence) {
	s.presenceMu.Lock()
	s.presence = presence
	s.presenceMu.Unlock()
}

func (s *C2S) Presence() *xmpp.Presence {
	s.presenceMu.RLock()
	defer s.presenceMu.RUnlock()
	return s.presence
}

func (s *C2S) Disconnect(err error) {
}

func (s *C2S) SendElement(elem xmpp.XElement) {
	stanza, ok := elem.(xmpp.Stanza)
	if !ok {
		return
	}
	s.cluster.SendMessageTo(s.node, &Message{
		Type: MsgRouteStanza,
		Node: s.cluster.LocalNode(),
		Payloads: []MessagePayload{{
			JID:    stanza.ToJID(),
			Stanza: stanza,
		}},
	})
}
