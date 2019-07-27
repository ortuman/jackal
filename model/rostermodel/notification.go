/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package rostermodel

import (
	"bytes"
	"encoding/gob"

	"github.com/ortuman/jackal/xmpp"
)

// Notification represents a roster subscription pending notification.
type Notification struct {
	Contact  string
	JID      string
	Presence *xmpp.Presence
}

// FromBytes deserializes a Notification entity from its binary representation.
func (rn *Notification) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&rn.Contact); err != nil {
		return err
	}
	if err := dec.Decode(&rn.JID); err != nil {
		return err
	}
	p, err := xmpp.NewPresenceFromBytes(buf)
	if err != nil {
		return err
	}
	rn.Presence = p
	return nil
}

// ToBytes converts a Notification entity to its binary representation.
func (rn *Notification) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&rn.Contact); err != nil {
		return err
	}
	if err := enc.Encode(&rn.JID); err != nil {
		return err
	}
	return rn.Presence.ToBytes(buf)
}
