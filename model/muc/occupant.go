/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"encoding/gob"

	"github.com/ortuman/jackal/xmpp/jid"
)

type Occupant struct {
	OccupantJID *jid.JID
	Nick        string
	FullJID     *jid.JID
	Affiliation string
	Role        string
}

// FromBytes deserializes an Occupant entity from it's gob binary representation.
func (o *Occupant) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	j, err := jid.NewFromBytes(buf)
	if err != nil {
		return err
	}
	o.OccupantJID = j
	if err := dec.Decode(&o.Nick); err != nil {
		return err
	}
	f, err := jid.NewFromBytes(buf)
	if err != nil {
		return err
	}
	o.FullJID = f
	if err := dec.Decode(&o.Affiliation); err != nil {
		return err
	}
	if err := dec.Decode(&o.Role); err != nil {
		return err
	}
	return nil
}

// ToBytes converts an Occupant entity to it's gob binary representation.
func (o *Occupant) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := o.OccupantJID.ToBytes(buf); err != nil {
		return err
	}
	if err := enc.Encode(&o.Nick); err != nil {
		return err
	}
	if err := o.FullJID.ToBytes(buf); err != nil {
		return err
	}
	if err := enc.Encode(&o.Affiliation); err != nil {
		return err
	}
	if err := enc.Encode(&o.Role); err != nil {
		return err
	}
	return nil
}

// NewOccupantFromBytes creates and returns a new Occupant element from its bytes representation.
func NewOccupantFromBytes(buf *bytes.Buffer) (*Occupant, error) {
	o := &Occupant{}
	if err := o.FromBytes(buf); err != nil {
		return nil, err
	}
	return o, nil
}
