/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"encoding/gob"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	msgBindType           byte = 1 << 0
	msgUnbindType         byte = 1 << 2
	msgUpdatePresenceType byte = 1 << 3
	msgRouteStanzaType    byte = 1 << 4
)

type baseMessage struct {
	Node string
	JID  *jid.JID
}

func (bm *baseMessage) FromGob(dec *gob.Decoder) error {
	dec.Decode(&bm.Node)
	j, err := jid.NewFromGob(dec)
	if err != nil {
		return err
	}
	bm.JID = j
	return nil
}

func (bm *baseMessage) ToGob(enc *gob.Encoder) {
	enc.Encode(bm.Node)
	bm.JID.ToGob(enc)
}

type BindMessage struct {
	baseMessage
}

type UnbindMessage struct {
	baseMessage
}

type UpdatePresenceMessage struct {
	baseMessage
	Presence *xmpp.Presence
}

func (bpm *UpdatePresenceMessage) FromGob(dec *gob.Decoder) error {
	bpm.baseMessage.FromGob(dec)
	p, err := xmpp.NewPresenceFromGob(dec)
	if err != nil {
		return err
	}
	bpm.Presence = p
	return nil
}

func (bpm *UpdatePresenceMessage) ToGob(enc *gob.Encoder) {
	bpm.baseMessage.ToGob(enc)
	bpm.Presence.ToGob(enc)
}

type RouteStanzaMessage struct {
	baseMessage
	Stanza xmpp.Stanza
}

func (rsm *RouteStanzaMessage) FromGob(dec *gob.Decoder) error {
	rsm.baseMessage.FromGob(dec)
	s, err := xmpp.NewStanzaFromElement(xmpp.NewElementFromGob(dec))
	if err != nil {
		return err
	}
	rsm.Stanza = s
	return nil
}

func (rsm *RouteStanzaMessage) ToGob(enc *gob.Encoder) {
	rsm.baseMessage.ToGob(enc)
	rsm.Stanza.ToGob(enc)
}
