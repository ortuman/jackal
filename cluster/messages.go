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
	msgBindType byte = 1
	msgUnbindType
	msgUpdatePresenceType
	msgSendStanzaType
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
	bpm.FromGob(dec)
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

type sendStanzaMessage struct {
	baseMessage
	Stanza xmpp.Stanza
}

func (ssm *sendStanzaMessage) FromGob(dec *gob.Decoder) error {
	ssm.baseMessage.FromGob(dec)
	s, err := xmpp.NewStanzaFromElement(xmpp.NewElementFromGob(dec))
	if err != nil {
		return err
	}
	ssm.Stanza = s
	return nil
}

func (ssm *sendStanzaMessage) ToGob(enc *gob.Encoder) {
	ssm.baseMessage.ToGob(enc)
	ssm.Stanza.ToGob(enc)
}
