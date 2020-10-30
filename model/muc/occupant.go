/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/ortuman/jackal/xmpp/jid"
)

const (
	// Affiliations
	member  = "member"
	admin   = "admin"
	owner   = "owner"
	outcast = "outcast"

	// Roles
	moderator   = "moderator"
	participant = "participant"
	visitor     = "visitor"
	none        = "none"
)

type Occupant struct {
	OccupantJID *jid.JID
	BareJID     *jid.JID
	affiliation string
	role        string
	// a set of different resources that the user uses to access this occupant
	resources map[string]bool
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
	var numResources int
	if err := dec.Decode(&numResources); err != nil {
		return err
	}
	o.resources = make(map[string]bool)
	for i := 0; i < numResources; i++ {
		var res string
		if err := dec.Decode(&res); err != nil {
			return err
		}
		o.resources[res] = true
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
	if err := enc.Encode(len(o.resources)); err != nil {
		return err
	}
	for res, _ := range o.resources {
		if err := enc.Encode(&res); err != nil {
			return err
		}
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

func NewOccupant(occJID, userJID *jid.JID) (*Occupant, error) {
	if !occJID.IsFullWithUser() {
		return nil, fmt.Errorf("Occupant JID %s is not valid", occJID.String())
	}
	if !userJID.IsBare() {
		return nil, fmt.Errorf("User JID %s is not a bare JID", userJID.String())
	}
	o := &Occupant{
		OccupantJID: occJID,
		BareJID:     userJID,
	}
	o.resources = make(map[string]bool)
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

func (o *Occupant) HasNoRole() bool {
	return o.role == None
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

func (o *Occupant) HasNoAffiliation() bool {
	return o.affiliation == None
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

func (o *Occupant) GetAllResources() []string {
	resources := make([]string, 0, len(o.resources))
	for r := range o.resources {
		resources = append(resources, r)
	}
	return resources
}

func (o *Occupant) HasResource(s string) bool {
	_, found := o.resources[s]
	return found
}

func (o *Occupant) AddResource(s string) {
	o.resources[s] = true
}

func (o *Occupant) DeleteResource(s string) {
	delete(o.resources, s)
}

func (o *Occupant) HasHigherAffiliation(k *Occupant) bool {
	switch {
	case o.IsOwner() :
		return true
	case o.IsAdmin() :
		return !k.IsOwner()
	case o.IsMember() :
		return !k.IsOwner() && !k.IsAdmin()
	case o.HasNoAffiliation() :
		return k.HasNoAffiliation()
	}
	return false
}

func (o *Occupant) CanChangeRole(target *Occupant, role string) bool {
	switch role {
	case none:
		return o.IsModerator() && o.HasHigherAffiliation(target)
	case visitor:
		return o.IsModerator() && target.IsParticipant()
	case participant:
		return o.IsModerator() && target.IsVisitor() || o.IsAdmin() && !target.IsOwner()
	case moderator:
		return o.IsAdmin() || o.IsOwner()
	}
	return false
}

func (o *Occupant) CanChangeAffiliation(target *Occupant, affiliation string) bool {
	if o.OccupantJID.String() == target.OccupantJID.String() {
		return false
	}
	if !o.IsAdmin() && !o.IsOwner() {
		return false
	}
	switch affiliation {
	case none:
		return o.HasHigherAffiliation(target)
	case member:
		return o.HasHigherAffiliation(target)
	case admin:
		return o.IsOwner()
	case owner:
		return o.IsOwner()
	}
	return false
}
