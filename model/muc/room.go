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

type Room struct {
	Config *RoomConfig
	// TODO Show name in the discovery instead of the default "Chatroom"
	Name           string
	RoomJID        *jid.JID
	Desc           string
	Subject        string
	Language       string
	OccupantsCnt   int
	NickToOccupant map[string]*Occupant //mapping nick in the room to the occupant
	UserToOccupant map[string]*Occupant //mapping user bare jid to the occupant
	Locked         bool
}

// FromBytes deserializes a Room entity from it's gob binary representation.
func (r *Room) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&r.Name); err != nil {
		return err
	}
	j, err := jid.NewFromBytes(buf)
	if err != nil {
		return err
	}
	r.RoomJID = j
	if err := dec.Decode(&r.Desc); err != nil {
		return err
	}
	if err := dec.Decode(&r.Subject); err != nil {
		return err
	}
	if err := dec.Decode(&r.Language); err != nil {
		return err
	}
	c, err := NewConfigFromBytes(buf)
	if err != nil {
		return err
	}
	r.Config = c
	if err := dec.Decode(&r.OccupantsCnt); err != nil {
		return err
	}
	r.NickToOccupant = make(map[string]*Occupant)
	r.UserToOccupant = make(map[string]*Occupant)
	for i := 0; i < r.OccupantsCnt; i++ {
		o, err := NewOccupantFromBytes(buf)
		if err != nil {
			return err
		}
		r.NickToOccupant[o.Nick] = o
		r.UserToOccupant[o.FullJID.ToBareJID().String()] = o
	}
	if err := dec.Decode(&r.Locked); err != nil {
		return err
	}
	return nil
}

// ToBytes converts a Room entity to it's gob binary representation.
func (r *Room) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&r.Name); err != nil {
		return err
	}
	if err := r.RoomJID.ToBytes(buf); err != nil {
		return err
	}
	if err := enc.Encode(&r.Desc); err != nil {
		return err
	}
	if err := enc.Encode(&r.Subject); err != nil {
		return err
	}
	if err := enc.Encode(&r.Language); err != nil {
		return err
	}
	if err := r.Config.ToBytes(buf); err != nil {
		return err
	}
	if err := enc.Encode(&r.OccupantsCnt); err != nil {
		return err
	}
	for _, occ := range r.NickToOccupant {
		if err := occ.ToBytes(buf); err != nil {
			return err
		}
	}
	if err := enc.Encode(&r.Locked); err != nil {
		return err
	}
	return nil
}

func (r *Room) GetAdmins() (admins []string) {
	for jid, occ := range r.UserToOccupant {
		if occ.Affiliation == Admin {
			admins = append(admins, jid)
		}
	}
	return
}

func (r *Room) GetOwners() (owners []string) {
	for jid, occ := range r.UserToOccupant {
		if occ.Affiliation == Owner {
			owners = append(owners, jid)
		}
	}
	return
}
