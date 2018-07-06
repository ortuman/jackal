/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"encoding/gob"
	"time"

	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
)

// User represents a user storage entity.
type User struct {
	Username       string
	Password       string
	LastPresence   *xml.Presence
	LastPresenceAt time.Time
}

// FromGob deserializes a User entity from it's gob binary representation.
func (u *User) FromGob(dec *gob.Decoder) {
	dec.Decode(&u.Username)
	dec.Decode(&u.Password)
	var hasPresence bool
	dec.Decode(&hasPresence)
	if hasPresence {
		p := &xml.Presence{}
		p.FromGob(dec)
		fromJID, _ := jid.NewWithString(p.From(), true)
		toJID, _ := jid.NewWithString(p.To(), true)
		p.SetFromJID(fromJID)
		p.SetToJID(toJID)
		u.LastPresence = p
		dec.Decode(&u.LastPresenceAt)
	}
}

// ToGob converts a User entity to it's gob binary representation.
func (u *User) ToGob(enc *gob.Encoder) {
	enc.Encode(&u.Username)
	enc.Encode(&u.Password)
	hasPresence := u.LastPresence != nil
	enc.Encode(&hasPresence)
	if hasPresence {
		u.LastPresence.ToGob(enc)
		u.LastPresenceAt = time.Now()
		enc.Encode(&u.LastPresenceAt)
	}
}
