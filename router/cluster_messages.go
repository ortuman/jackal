/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"encoding/gob"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	msgBindType byte = 1
	msgUnbindType
	msgBroadcastPresenceType
	msgSendStanzaType
)

type baseMessage struct {
	node string
	jid  *jid.JID
}

func (bm *baseMessage) FromGob(dec *gob.Decoder) error {
	dec.Decode(&bm.node)
	j, err := jid.NewFromGob(dec)
	if err != nil {
		return err
	}
	bm.jid = j
	return nil
}

func (bm *baseMessage) ToGob(enc *gob.Encoder) {
	enc.Encode(bm.node)
	bm.jid.ToGob(enc)
}

type bindMessage struct {
	baseMessage
}

func newBindMessage(node string, jid *jid.JID) *bindMessage {
	return &bindMessage{baseMessage{node: node, jid: jid}}
}

type unbindMessage struct {
	baseMessage
}

func newUnbindMessage(node string, jid *jid.JID) *unbindMessage {
	return &unbindMessage{baseMessage{node: node, jid: jid}}
}

type broadcastPresenceMessage struct {
	baseMessage
	presence *xmpp.Presence
}

func newPresenceMessage(node string, jid *jid.JID, presence *xmpp.Presence) *broadcastPresenceMessage {
	return &broadcastPresenceMessage{baseMessage{node: node, jid: jid}, presence}
}

func (bpm *broadcastPresenceMessage) FromGob(dec *gob.Decoder) error {
	bpm.FromGob(dec)
	p, err := xmpp.NewPresenceFromGob(dec)
	if err != nil {
		return err
	}
	bpm.presence = p
	return nil
}

func (bpm *broadcastPresenceMessage) ToGob(enc *gob.Encoder) {
	bpm.baseMessage.ToGob(enc)
	bpm.presence.ToGob(enc)
}

type sendStanzaMessage struct {
	baseMessage
	stanza xmpp.Stanza
}

func newSendStanzaMessage(node string, jid *jid.JID, stanza xmpp.Stanza) *sendStanzaMessage {
	return &sendStanzaMessage{baseMessage{node: node, jid: jid}, stanza}
}

func (ssm *sendStanzaMessage) FromGob(dec *gob.Decoder) error {
	ssm.baseMessage.FromGob(dec)
	s, err := xmpp.NewStanzaFromElement(xmpp.NewElementFromGob(dec))
	if err != nil {
		return err
	}
	ssm.stanza = s
	return nil
}

func (ssm *sendStanzaMessage) ToGob(enc *gob.Encoder) {
	ssm.baseMessage.ToGob(enc)
	ssm.stanza.ToGob(enc)
}
