/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package cluster

import (
	"bytes"
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

// FromBytes reads MessagePayload fields from its binary representation.
func (p *MessagePayload) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	j, err := jid.NewFromBytes(buf)
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
		message, err := xmpp.NewMessageFromBytes(buf)
		if err != nil {
			return err
		}
		p.Stanza = message
	case presenceStanza:
		presence, err := xmpp.NewPresenceFromBytes(buf)
		if err != nil {
			return err
		}
		p.Stanza = presence
	case iqStanza:
		iq, err := xmpp.NewIQFromBytes(buf)
		if err != nil {
			return err
		}
		p.Stanza = iq
	}
	return nil
}

// ToBytes converts a MessagePayload instance to its binary representation.
func (p *MessagePayload) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := p.JID.ToBytes(buf); err != nil {
		return err
	}

	hasContextMap := p.Context != nil
	if err := enc.Encode(&hasContextMap); err != nil {
		return err
	}
	if hasContextMap {
		if err := enc.Encode(&p.Context); err != nil {
			return err
		}
	}

	hasStanza := p.Stanza != nil
	if err := enc.Encode(&hasStanza); err != nil {
		return err
	}
	if !hasStanza {
		return nil
	}
	// store stanza type
	switch p.Stanza.(type) {
	case *xmpp.Message:
		if err := enc.Encode(messageStanza); err != nil {
			return err
		}
	case *xmpp.Presence:
		if err := enc.Encode(presenceStanza); err != nil {
			return err
		}
	case *xmpp.IQ:
		if err := enc.Encode(iqStanza); err != nil {
			return err
		}
	default:
		return nil
	}
	return p.Stanza.ToBytes(buf)
}

// Message is the c2s message type.
// A message can contain one or more payloads.
type Message struct {
	Type     int
	Node     string
	Payloads []MessagePayload
}

// FromBytes reads Message fields from its binary representation.
func (m *Message) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&m.Type); err != nil {
		return err
	}
	if err := dec.Decode(&m.Node); err != nil {
		return err
	}

	var pLen int
	if err := dec.Decode(&pLen); err != nil {
		return err
	}

	m.Payloads = nil
	for i := 0; i < pLen; i++ {
		var p MessagePayload
		if err := p.FromBytes(buf); err != nil {
			return err
		}
		m.Payloads = append(m.Payloads, p)
	}
	return nil
}

// ToBytes converts a Message instance to its binary representation.
func (m *Message) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(m.Type); err != nil {
		return err
	}
	if err := enc.Encode(m.Node); err != nil {
		return err
	}
	if err := enc.Encode(len(m.Payloads)); err != nil {
		return err
	}
	for _, p := range m.Payloads {
		if err := p.ToBytes(buf); err != nil {
			return err
		}
	}
	return nil
}
