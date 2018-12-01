package cluster

import (
	"bytes"
	"encoding/gob"
	"sync"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const c2sMailboxSize = 4096

type C2S struct {
	identifier string
	buf        *bytes.Buffer
	cluster    *Cluster
	node       string
	jid        *jid.JID
	presenceMu sync.RWMutex
	presence   *xmpp.Presence
	actorCh    chan func()
}

func newC2S(identifier string, jid *jid.JID, node string, cluster *Cluster) *C2S {
	s := &C2S{
		identifier: identifier,
		buf:        bytes.NewBuffer(nil),
		cluster:    cluster,
		node:       node,
		jid:        jid,
		presence:   xmpp.NewPresence(jid, jid, xmpp.UnavailableType),
		actorCh:    make(chan func(), c2sMailboxSize),
	}
	go s.loop()
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
func (s *C2S) IsCompressed() bool    { return false }

func (s *C2S) Presence() *xmpp.Presence {
	s.presenceMu.RLock()
	defer s.presenceMu.RUnlock()
	return s.presence
}

func (s *C2S) Disconnect(err error) {
}

func (s *C2S) SendElement(elem xmpp.XElement) {
	s.actorCh <- func() {
		s.sendElement(elem)
	}
}

func (s *C2S) sendElement(elem xmpp.XElement) {
	stanza, ok := elem.(xmpp.Stanza)
	if !ok {
		return
	}
	defer s.buf.Reset()

	msg := &RouteStanzaMessage{baseMessage{
		Node: s.cluster.LocalNode(),
		JID:  stanza.ToJID()},
		stanza,
	}
	enc := gob.NewEncoder(s.buf)
	msg.ToGob(enc)
	if err := s.cluster.Send(s.buf.Bytes(), s.node); err != nil {
		log.Error(err)
	}
}

func (s *C2S) loop() {
	for f := range s.actorCh {
		f()
	}
}

func (s *C2S) setPresence(presence *xmpp.Presence) {
	s.presenceMu.Lock()
	s.presence = presence
	s.presenceMu.Unlock()
}

func (s *C2S) close() {
	close(s.actorCh)
}
