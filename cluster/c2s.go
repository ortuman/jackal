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

type c2sCluster interface {
	LocalNode() string
	SendMessageTo(node string, msg *Message)
}

// C2S represents a cluster c2s stream.
type C2S struct {
	identifier string
	cluster    c2sCluster
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
	cluster c2sCluster) *C2S {
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

// ID returns stream identifier.
func (s *C2S) ID() string {
	return s.identifier
}

// Context returns a copy of the stream associated context.
func (s *C2S) Context() map[string]interface{} {
	m := make(map[string]interface{})
	s.contextMu.RLock()
	for k, v := range s.context {
		m[k] = v
	}
	s.contextMu.RUnlock()
	return m
}

// SetString associates a string context value to a key.
func (s *C2S) SetString(key string, value string) {}

// GetString returns the context value associated with the key as a string.
func (s *C2S) GetString(key string) string {
	var ret string
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if s, ok := s.context[key].(string); ok {
		ret = s
	}
	return ret
}

// SetInt associates an integer context value to a key.
func (s *C2S) SetInt(key string, value int) {}

// GetInt returns the context value associated with the key as an integer.
func (s *C2S) GetInt(key string) int {
	var ret int
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if i, ok := s.context[key].(int); ok {
		ret = i
	}
	return ret
}

// SetFloat associates a float context value to a key.
func (s *C2S) SetFloat(key string, value float64) {}

// GetFloat returns the context value associated with the key as a float64.
func (s *C2S) GetFloat(key string) float64 {
	var ret float64
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if f, ok := s.context[key].(float64); ok {
		ret = f
	}
	return ret
}

// SetBool associates a boolean context value to a key.
func (s *C2S) SetBool(key string, value bool) {}

// GetBool returns the context value associated with the key as a boolean.
func (s *C2S) GetBool(key string) bool {
	var ret bool
	s.contextMu.RLock()
	defer s.contextMu.RUnlock()
	if b, ok := s.context[key].(bool); ok {
		ret = b
	}
	return ret
}

// UpdateContext updates stream context by copying all 'm' values
func (s *C2S) UpdateContext(m map[string]interface{}) {
	s.contextMu.Lock()
	for k, v := range m {
		s.context[k] = v
	}
	s.contextMu.Unlock()
}

// Username returns current stream username.
func (s *C2S) Username() string {
	return s.jid.Node()
}

// Domain returns current stream domain.
func (s *C2S) Domain() string {
	return s.jid.Domain()
}

// Resource returns current stream resource.
func (s *C2S) Resource() string {
	return s.jid.Resource()
}

// JID returns current user JID.
func (s *C2S) JID() *jid.JID {
	return s.jid
}

// IsAuthenticated returns whether or not the XMPP stream has successfully authenticated.
func (s *C2S) IsAuthenticated() bool { return true }

// IsSecured returns whether or not the XMPP stream has been secured using SSL/TLS.
func (s *C2S) IsSecured() bool { return true }

// Presence returns last sent presence element.
func (s *C2S) Presence() *xmpp.Presence {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.presence
}

// SetPresence updates the C2S stream presence.
func (s *C2S) SetPresence(presence *xmpp.Presence) {
	s.mu.Lock()
	s.presence = presence
	s.mu.Unlock()
}

// Disconnect disconnects remote peer by closing the underlying TCP socket connection.
func (s *C2S) Disconnect(err error) {}

// SendElement writes an XMPP element to the stream.
func (s *C2S) SendElement(elem xmpp.XElement) {
	stanza, ok := elem.(xmpp.Stanza)
	if !ok {
		return
	}
	s.cluster.SendMessageTo(s.node, &Message{
		Type: MsgRouteStanza,
		Node: s.cluster.LocalNode(),
		Payloads: []MessagePayload{{
			JID:    s.jid,
			Stanza: stanza,
		}},
	})
}
