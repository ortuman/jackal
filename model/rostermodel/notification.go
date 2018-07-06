/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package rostermodel

import (
	"encoding/gob"

	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
)

// Notification represents a roster subscription
// pending notification.
type Notification struct {
	Contact  string
	JID      string
	Presence *xml.Presence
}

// FromGob deserializes a Notification entity
// from it's gob binary representation.
func (rn *Notification) FromGob(dec *gob.Decoder) {
	dec.Decode(&rn.Contact)
	dec.Decode(&rn.JID)
	el := &xml.Element{}
	el.FromGob(dec)
	fromJID, _ := jid.NewWithString(el.From(), true)
	toJID, _ := jid.NewWithString(el.To(), true)
	rn.Presence, _ = xml.NewPresenceFromElement(el, fromJID, toJID)
}

// ToGob converts a Notification entity
// to it's gob binary representation.
func (rn *Notification) ToGob(enc *gob.Encoder) {
	enc.Encode(&rn.Contact)
	enc.Encode(&rn.JID)
	rn.Presence.ToGob(enc)
}
