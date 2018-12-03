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
	MsgBindType = iota
	MsgUnbindType
	MsgUpdatePresenceType
	MsgRouteStanzaType
)

const (
	messageStanzaType = iota
	presenceStanzaType
	iqStanzaType
)

type Message struct {
	Type   int
	Node   string
	JIDs   []*jid.JID
	Stanza xmpp.Stanza
}

func (m *Message) FromGob(dec *gob.Decoder) error {
	var jLen int
	dec.Decode(&m.Type)
	dec.Decode(&m.Node)
	dec.Decode(&jLen)
	if jLen > 0 {
		j, err := jid.NewFromGob(dec)
		if err != nil {
			return err
		}
		m.JIDs = append(m.JIDs, j)
	}
	var hasStanza bool
	dec.Decode(&hasStanza)
	if hasStanza {
		var stanzaType int
		dec.Decode(&stanzaType)
		switch stanzaType {
		case messageStanzaType:
			message, err := xmpp.NewMessageFromGob(dec)
			if err != nil {
				return err
			}
			m.Stanza = message
		case presenceStanzaType:
			presence, err := xmpp.NewMessageFromGob(dec)
			if err != nil {
				return err
			}
			m.Stanza = presence
		case iqStanzaType:
			iq, err := xmpp.NewMessageFromGob(dec)
			if err != nil {
				return err
			}
			m.Stanza = iq
		default:
			return nil
		}
	}
	return nil
}

func (m *Message) ToGob(enc *gob.Encoder) {
	enc.Encode(m.Type)
	enc.Encode(m.Node)
	jLen := len(m.JIDs)
	enc.Encode(jLen)
	if jLen > 0 {
		for _, j := range m.JIDs {
			j.ToGob(enc)
		}
	}
	hasStanza := m.Stanza != nil
	enc.Encode(&hasStanza)
	if hasStanza {
		// store stanza type
		switch m.Stanza.(type) {
		case *xmpp.Message:
			enc.Encode(messageStanzaType)
		case *xmpp.Presence:
			enc.Encode(presenceStanzaType)
		case *xmpp.IQ:
			enc.Encode(iqStanzaType)
		default:
			return
		}
		m.Stanza.ToGob(enc)
	}
}
