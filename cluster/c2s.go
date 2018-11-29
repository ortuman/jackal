package cluster

import (
	"sync"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type C2S struct {
	cluster    Cluster
	node       string
	identifier string
	mu         sync.RWMutex
	jid        *jid.JID
	presence   *xmpp.Presence
}

func (s *C2S) ID() string {
	return s.identifier
}

func (s *C2S) Context() *stream.Context {
	return nil
}

func (s *C2S) Username() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jid.Node()
}

func (s *C2S) Domain() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jid.Domain()
}

func (s *C2S) Resource() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jid.Resource()
}

func (s *C2S) JID() *jid.JID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jid
}

func (s *C2S) IsSecured() bool       { return true }
func (s *C2S) IsAuthenticated() bool { return true }
func (s *C2S) IsCompressed() bool    { return false }

func (s *C2S) Presence() *xmpp.Presence {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.presence
}

func (s *C2S) Disconnect(err error) {
	// This stream actually does not belong to this Node.
	// Performing a disconnect should do anything.
}

func (s *C2S) SendElement(elem xmpp.XElement) {
}
