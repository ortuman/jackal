/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"sync"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type C2S struct {
	identifier string
	cluster    *Cluster
	node       string
	jid        *jid.JID
	mu         sync.RWMutex
	presence   *xmpp.Presence
	contextMu  sync.RWMutex
	context    map[string]interface{}
}

func newC2S(
	identifier string,
	jid *jid.JID,
	presence *xmpp.Presence,
	context map[string]interface{},
	node string,
	cluster *Cluster) *C2S {
	s := &C2S{
		identifier: identifier,
		cluster:    cluster,
		node:       node,
		jid:        jid,
		presence:   presence,
		context:    context,
	}
	return s
}

func (s *C2S) ID() string {
	return s.identifier
}

func (s *C2S) Context() map[string]interface{} {
	m := make(map[string]interface{})
	s.contextMu.RLock()
	for k, v := range s.context {
		m[k] = v
	}
	s.contextMu.RUnlock()
	return m
}

func (s *C2S) SetString(key string, value string) {}

func (s *C2S) GetString(key string) string {
	var ret string
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if s, ok := s.context[key].(string); ok {
		ret = s
	}
	return ret
}

func (s *C2S) SetInt(key string, value int) {}

func (s *C2S) GetInt(key string) int {
	var ret int
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if i, ok := s.context[key].(int); ok {
		ret = i
	}
	return ret
}

func (s *C2S) SetFloat(key string, value float64) {}

func (s *C2S) GetFloat(key string) float64 {
	var ret float64
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if f, ok := s.context[key].(float64); ok {
		ret = f
	}
	return ret
}

func (s *C2S) SetBool(key string, value bool) {}

func (s *C2S) GetBool(key string) bool {
	var ret bool
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if b, ok := s.context[key].(bool); ok {
		ret = b
	}
	return ret
}

func (s *C2S) UpdateContext(m map[string]interface{}) {
	s.contextMu.Lock()
	for k, v := range m {
		s.context[k] = v
	}
	s.contextMu.Unlock()
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
	s.mu.Lock()
	s.presence = presence
	s.mu.Unlock()
}

func (s *C2S) Presence() *xmpp.Presence {
	s.mu.RLock()
	defer s.mu.RUnlock()
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
