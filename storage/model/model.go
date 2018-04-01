/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"encoding/gob"

	"github.com/ortuman/jackal/xml"
)

// User represents a user storage entity.
type User struct {
	Username string
	Password string
}

// FromBytes deserializes a User entity
// from it's gob binary representation.
func (u *User) FromGob(dec *gob.Decoder) {
	dec.Decode(&u.Username)
	dec.Decode(&u.Password)
}

// ToBytes converts a User entity
// to it's gob binary representation.
func (u *User) ToGob(enc *gob.Encoder) {
	enc.Encode(&u.Username)
	enc.Encode(&u.Password)
}

// RosterItem represents a roster item storage entity.
type RosterItem struct {
	User         string
	Contact      string
	Name         string
	Subscription string
	Ask          bool
	Groups       []string
}

// FromBytes deserializes a RosterItem entity
// from it's gob binary representation.
func (ri *RosterItem) FromGob(dec *gob.Decoder) {
	dec.Decode(&ri.User)
	dec.Decode(&ri.Contact)
	dec.Decode(&ri.Name)
	dec.Decode(&ri.Subscription)
	dec.Decode(&ri.Ask)
	dec.Decode(&ri.Groups)
}

// ToBytes converts a RosterItem entity
// to it's gob binary representation.
func (ri *RosterItem) ToGob(enc *gob.Encoder) {
	enc.Encode(&ri.User)
	enc.Encode(&ri.Contact)
	enc.Encode(&ri.Name)
	enc.Encode(&ri.Subscription)
	enc.Encode(&ri.Ask)
	enc.Encode(&ri.Groups)
}

// RosterNotification represents a roster subscription
// pending notification.
type RosterNotification struct {
	User     string
	Contact  string
	Elements []xml.XElement
}

// FromGob deserializes a RosterNotification entity
// from it's gob binary representation.
func (rn *RosterNotification) FromGob(dec *gob.Decoder) {
	dec.Decode(&rn.User)
	dec.Decode(&rn.Contact)
	var ln int
	dec.Decode(&ln)
	for i := 0; i < ln; i++ {
		rn.Elements = append(rn.Elements, xml.NewElementFromGob(dec))
	}
}

// ToGob converts a RosterNotification entity
// to it's gob binary representation.
func (rn *RosterNotification) ToGob(enc *gob.Encoder) {
	enc.Encode(&rn.User)
	enc.Encode(&rn.Contact)
	enc.Encode(len(rn.Elements))
	for _, el := range rn.Elements {
		el.ToGob(enc)
	}
}
