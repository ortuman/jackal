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
	MsgUnbind
	MsgUpdatePresence
	MsgRouteStanza
)

const (
	messageStanza = iota
	presenceStanza
	iqStanza
)

type clusterPackage struct {
	messages []Message
}

func (p *clusterPackage) FromGob(dec *gob.Decoder) error {
	var ln int
	_ = dec.Decode(&ln)
	p.messages = make([]Message, 0, ln)
	for i := 0; i < ln; i++ {
		var msg Message
		_ = msg.FromGob(dec)
		p.messages = append(p.messages, msg)
	}
	return nil
}

func (p *clusterPackage) ToGob(enc *gob.Encoder) {
	_ = enc.Encode(len(p.messages))
	for _, m := range p.messages {
		m.ToGob(enc)
	}
}

type Message struct {
	Type   int
	Node   string
	JID    *jid.JID
	Stanza xmpp.Stanza
}

func (m *Message) FromGob(dec *gob.Decoder) error {
	_ = dec.Decode(&m.Type)
	_ = dec.Decode(&m.Node)
	j, err := jid.NewFromGob(dec)
	if err != nil {
		return err
	}
	m.JID = j

	var hasStanza bool
	_ = dec.Decode(&hasStanza)
	if !hasStanza {
		return nil
	}
	var stanzaType int
	_ = dec.Decode(&stanzaType)
	switch stanzaType {
	case messageStanza:
		message, err := xmpp.NewMessageFromGob(dec)
		if err != nil {
			return err
		}
		m.Stanza = message
	case presenceStanza:
		presence, err := xmpp.NewMessageFromGob(dec)
		if err != nil {
			return err
		}
		m.Stanza = presence
	case iqStanza:
		iq, err := xmpp.NewMessageFromGob(dec)
		if err != nil {
			return err
		}
		m.Stanza = iq
	default:
		break
	}
	return nil
}

func (m *Message) ToGob(enc *gob.Encoder) {
	_ = enc.Encode(m.Type)
	_ = enc.Encode(m.Node)
	m.JID.ToGob(enc)
	hasStanza := m.Stanza != nil
	_ = enc.Encode(&hasStanza)
	if !hasStanza {
		return
	}
	// store stanza type
	switch m.Stanza.(type) {
	case *xmpp.Message:
		_ = enc.Encode(messageStanza)
	case *xmpp.Presence:
		_ = enc.Encode(presenceStanza)
	case *xmpp.IQ:
		_ = enc.Encode(iqStanza)
	default:
		return
	}
	m.Stanza.ToGob(enc)
}
