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
	// MsgBind represents a bind cluster message.
	MsgBind = iota

	// MsgBatchBind represents a batch bind cluster message.
	MsgBatchBind

	// MsgUnbind represents a unbind cluster message.
	MsgUnbind

	// MsgUpdatePresence represents an update presence cluster message.
	MsgUpdatePresence

	// MsgUpdateContext represents a context update cluster message.
	MsgUpdateContext

	// MsgRouteStanza represents a route stanza cluster message.
	MsgRouteStanza
)

const (
	messageStanza = iota
	presenceStanza
	iqStanza
)

// MessagePayload represents a message payload type.
type MessagePayload struct {
	JID     *jid.JID
	Context map[string]interface{}
	Stanza  xmpp.Stanza
}

// FromGob reads MessagePayload fields from its gob binary representation.
func (p *MessagePayload) FromGob(dec *gob.Decoder) error {
	j, err := jid.NewFromGob(dec)
	if err != nil {
		return err
	}
	p.JID = j

	var hasContextMap bool
	dec.Decode(&hasContextMap)
	if hasContextMap {
		var m map[string]interface{}
		dec.Decode(&m)
		p.Context = m
	}

	var hasStanza bool
	dec.Decode(&hasStanza)
	if !hasStanza {
		return nil
	}
	var stanzaType int
	dec.Decode(&stanzaType)
	switch stanzaType {
	case messageStanza:
		message, err := xmpp.NewMessageFromGob(dec)
		if err != nil {
			return err
		}
		p.Stanza = message
	case presenceStanza:
		presence, err := xmpp.NewPresenceFromGob(dec)
		if err != nil {
			return err
		}
		p.Stanza = presence
	case iqStanza:
		iq, err := xmpp.NewIQFromGob(dec)
		if err != nil {
			return err
		}
		p.Stanza = iq
	}
	return nil
}

// ToGob converts a MessagePayload instance to its gob binary representation.
func (p *MessagePayload) ToGob(enc *gob.Encoder) {
	p.JID.ToGob(enc)

	hasContextMap := p.Context != nil
	enc.Encode(&hasContextMap)
	if hasContextMap {
		enc.Encode(&p.Context)
	}

	hasStanza := p.Stanza != nil
	enc.Encode(&hasStanza)
	if !hasStanza {
		return
	}
	// store stanza type
	switch p.Stanza.(type) {
	case *xmpp.Message:
		enc.Encode(messageStanza)
	case *xmpp.Presence:
		enc.Encode(presenceStanza)
	case *xmpp.IQ:
		enc.Encode(iqStanza)
	default:
		return
	}
	p.Stanza.ToGob(enc)
}

// Message is the cluster message type.
// A message can contain one or more payloads.
type Message struct {
	Type     int
	Node     string
	Payloads []MessagePayload
}

// FromGob reads Message fields from its gob binary representation.
func (m *Message) FromGob(dec *gob.Decoder) error {
	dec.Decode(&m.Type)
	dec.Decode(&m.Node)

	var pLen int
	dec.Decode(&pLen)

	m.Payloads = nil
	for i := 0; i < pLen; i++ {
		var p MessagePayload
		p.FromGob(dec)
		m.Payloads = append(m.Payloads, p)
	}
	return nil
}

// ToGob converts a Message instance to its gob binary representation.
func (m *Message) ToGob(enc *gob.Encoder) {
	enc.Encode(m.Type)
	enc.Encode(m.Node)
	enc.Encode(len(m.Payloads))
	for _, p := range m.Payloads {
		p.ToGob(enc)
	}
}
