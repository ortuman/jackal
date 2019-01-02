/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"encoding/gob"
	"time"

	"github.com/ortuman/jackal/xmpp"
)

// User represents a user storage entity.
type User struct {
	Username       string
	Password       string
	LastPresence   *xmpp.Presence
	LastPresenceAt time.Time
}

// FromGob deserializes a User entity from it's gob binary representation.
func (u *User) FromGob(dec *gob.Decoder) error {
	dec.Decode(&u.Username)
	dec.Decode(&u.Password)
	var hasPresence bool
	dec.Decode(&hasPresence)
	if hasPresence {
		p, err := xmpp.NewPresenceFromGob(dec)
		if err != nil {
			return err
		}
		u.LastPresence = p
		dec.Decode(&u.LastPresenceAt)
	}
	return nil
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
