/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"encoding/gob"
	"time"

	"github.com/ortuman/jackal/xml"
)

// GobSerializer represents a Gob serializable entity.
type GobSerializer interface {
	ToGob(enc *gob.Encoder)
}

// GobDeserializer represents a Gob deserializable entity.
type GobDeserializer interface {
	FromGob(dec *gob.Decoder)
}

// User represents a user storage entity.
type User struct {
	Username        string
	Password        string
	LoggedOutStatus string
	LoggedOutAt     time.Time
}

// FromGob deserializes a User entity from it's gob binary representation.
func (u *User) FromGob(dec *gob.Decoder) {
	dec.Decode(&u.Username)
	dec.Decode(&u.Password)
	dec.Decode(&u.LoggedOutStatus)
	dec.Decode(&u.LoggedOutAt)
}

// ToBytes converts a User entity to it's gob binary representation.
func (u *User) ToGob(enc *gob.Encoder) {
	enc.Encode(&u.Username)
	enc.Encode(&u.Password)
	enc.Encode(&u.LoggedOutStatus)
	enc.Encode(&u.LoggedOutAt)
}

// RosterItem represents a roster item storage entity.
type RosterItem struct {
	User         string
	Contact      string
	Name         string
	Subscription string
	Ask          bool
	Ver          int
	Groups       []string
}

// FromGob deserializes a RosterItem entity
// from it's gob binary representation.
func (ri *RosterItem) FromGob(dec *gob.Decoder) {
	dec.Decode(&ri.User)
	dec.Decode(&ri.Contact)
	dec.Decode(&ri.Name)
	dec.Decode(&ri.Subscription)
	dec.Decode(&ri.Ask)
	dec.Decode(&ri.Ver)
	dec.Decode(&ri.Groups)
}

// ToGob converts a RosterItem entity
// to it's gob binary representation.
func (ri *RosterItem) ToGob(enc *gob.Encoder) {
	enc.Encode(&ri.User)
	enc.Encode(&ri.Contact)
	enc.Encode(&ri.Name)
	enc.Encode(&ri.Subscription)
	enc.Encode(&ri.Ask)
	enc.Encode(&ri.Ver)
	enc.Encode(&ri.Groups)
}

// RosterVersion represents a roster version info.
type RosterVersion struct {
	Ver         int
	DeletionVer int
}

// FromGob deserializes a RosterVersion entity
// from it's gob binary representation.
func (rv *RosterVersion) FromGob(dec *gob.Decoder) {
	dec.Decode(&rv.Ver)
	dec.Decode(&rv.DeletionVer)
}

// ToGob converts a RosterVersion entity
// to it's gob binary representation.
func (rv *RosterVersion) ToGob(enc *gob.Encoder) {
	enc.Encode(&rv.Ver)
	enc.Encode(&rv.DeletionVer)
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
		var e xml.Element
		e.FromGob(dec)
		rn.Elements = append(rn.Elements, &e)
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
