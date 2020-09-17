/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"fmt"
	"encoding/gob"

	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	// Affiliations
	member = "member"

	admin = "admin"

	owner = "owner"

	outcast = "outcast"

	// Roles
	moderator = "moderator"

	participant = "participant"

	visitor = "visitor"
)

type Occupant struct {
	OccupantJID *jid.JID
	BareJID     *jid.JID
	affiliation string
	role        string
}

// FromBytes deserializes an Occupant entity from it's gob binary representation.
func (o *Occupant) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	j, err := jid.NewFromBytes(buf)
	if err != nil {
		return err
	}
	o.OccupantJID = j
	f, err := jid.NewFromBytes(buf)
	if err != nil {
		return err
	}
	o.BareJID = f
	if err := dec.Decode(&o.affiliation); err != nil {
		return err
	}
	if err := dec.Decode(&o.role); err != nil {
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
	if err := o.BareJID.ToBytes(buf); err != nil {
		return err
	}
	if err := enc.Encode(&o.affiliation); err != nil {
		return err
	}
	if err := enc.Encode(&o.role); err != nil {
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

func (o *Occupant) SetAffiliation(aff string) error {
	switch aff {
	case owner, admin, member, outcast, None:
		o.affiliation = aff
	default:
		return fmt.Errorf("occupant: this type of affiliation is not supported - %s", aff)
	}
	return nil
}

func (o *Occupant) GetAffiliation() string {
	return o.affiliation
}

func (o *Occupant) SetRole(role string) error {
	switch role {
	case moderator, participant, visitor, None:
		o.role = role
	default:
		return fmt.Errorf("occupant: this type of role is not supported - %s", role)
	}
	return nil
}

func (o *Occupant) GetRole() string {
	return o.role
}

func (o *Occupant) IsVisitor() bool {
	return o.role == visitor
}

func (o *Occupant) IsParticipant() bool {
	return o.role == participant
}

func (o *Occupant) IsModerator() bool {
	return o.role == moderator
}

func (o *Occupant) IsOwner() bool {
	return o.affiliation == owner
}

func (o *Occupant) IsAdmin() bool {
	return o.affiliation == admin
}

func (o *Occupant) IsMember() bool {
	return o.affiliation == member
}

func (o *Occupant) IsOutcast() bool {
	return o.affiliation == outcast
}
