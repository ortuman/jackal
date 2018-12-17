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
	MsgBind = iota
	MsgBatchBind
	MsgUnbind
	MsgUpdatePresence
	MsgUpdateContext
	MsgRouteStanza
)

const (
	messageStanza = iota
	presenceStanza
	iqStanza
)

type MessagePayload struct {
	JID     *jid.JID
	Context map[string]interface{}
	Stanza  xmpp.Stanza
}

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

func (p *MessagePayload) ToGob(enc *gob.Encoder) {
	if p.JID == nil {
		panic("Oops")
	}
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

type Message struct {
	Type     int
	Node     string
	Payloads []MessagePayload
}

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

func (m *Message) ToGob(enc *gob.Encoder) {
	enc.Encode(m.Type)
	enc.Encode(m.Node)
	enc.Encode(len(m.Payloads))
	for _, p := range m.Payloads {
		p.ToGob(enc)
	}
}
