package router

import (
	"sync"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type clusterC2S struct {
	cluster    Cluster
	node       string
	identifier string
	mu         sync.RWMutex
	jid        *jid.JID
	presence   *xmpp.Presence
}

func (s *clusterC2S) ID() string {
	return s.identifier
}

func (s *clusterC2S) Context() *stream.Context {
	return nil
}

func (s *clusterC2S) Username() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jid.Node()
}

func (s *clusterC2S) Domain() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jid.Domain()
}

func (s *clusterC2S) Resource() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jid.Resource()
}

func (s *clusterC2S) JID() *jid.JID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jid
}

func (s *clusterC2S) IsSecured() bool       { return true }
func (s *clusterC2S) IsAuthenticated() bool { return true }
func (s *clusterC2S) IsCompressed() bool    { return false }

func (s *clusterC2S) Presence() *xmpp.Presence {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.presence
}

func (s *clusterC2S) Disconnect(err error) {
	// This stream actually does not belong to this node.
	// Performing a disconnect should do anything.
}

func (s *clusterC2S) SendElement(elem xmpp.XElement) {
}
