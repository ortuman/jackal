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

type messageType int

const (
	messageBindType messageType = iota
	messageUnbindType
	messageSendType
)

type baseMessage struct {
	node string
	jid  *jid.JID
}

func (bm *baseMessage) FromGob(dec *gob.Decoder) error {
	return nil
}

func (bm *baseMessage) ToGob(enc *gob.Encoder) {
}

type bindMessage struct {
	baseMessage
}

type unbindMessage struct {
	baseMessage
}

type broadcastPresenceMessage struct {
	baseMessage
	presence *xmpp.Presence
}

func (bpm *broadcastPresenceMessage) FromGob(dec *gob.Decoder) error {
	return nil
}

func (bpm *broadcastPresenceMessage) ToGob(enc *gob.Encoder) {
}

type sendStanzaMessage struct {
	baseMessage
	stanza xmpp.Stanza
}

func (ssm *sendStanzaMessage) FromGob(dec *gob.Decoder) error {
	return nil
}

func (ssm *sendStanzaMessage) ToGob(enc *gob.Encoder) {
}
