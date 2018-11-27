/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package rostermodel

import (
	"encoding/gob"

	"github.com/ortuman/jackal/xmpp"
)

// Notification represents a roster subscription pending notification.
type Notification struct {
	Contact  string
	JID      string
	Presence *xmpp.Presence
}

// FromGob deserializes a Notification entity from it's gob binary representation.
func (rn *Notification) FromGob(dec *gob.Decoder) error {
	dec.Decode(&rn.Contact)
	dec.Decode(&rn.JID)
	p, err := xmpp.NewPresenceFromGob(dec)
	if err != nil {
		return err
	}
	rn.Presence = p
	return nil
}

// ToGob converts a Notification entity
// to it's gob binary representation.
func (rn *Notification) ToGob(enc *gob.Encoder) {
	enc.Encode(&rn.Contact)
	enc.Encode(&rn.JID)
	rn.Presence.ToGob(enc)
}
