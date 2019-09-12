/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pubsubmodel

import (
	"bytes"
	"encoding/gob"
)

// affiliation definitions
const (
	Owner      = "owner"
	Subscriber = "subscriber"
)

// subscription definitions
const (
	None       = "none"
	Subscribed = "subscribed"
)

type Affiliation struct {
	JID         string
	Affiliation string
}

// FromBytes deserializes a Affiliation entity from its binary representation.
func (a *Affiliation) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&a.JID); err != nil {
		return err
	}
	return dec.Decode(&a.Affiliation)
}

// ToBytes converts a Affiliation entity to its binary representation.
func (a *Affiliation) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(a.JID); err != nil {
		return err
	}
	return enc.Encode(a.Affiliation)
}
