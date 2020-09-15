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
	Config            *RoomConfig
	Name              string
	RoomJID           *jid.JID
	Desc              string
	Subject           string
	Language          string
	Locked            bool
	numberOfOccupants int
	//mapping user bare jid to the occupant JID
	UserToOccupant map[jid.JID]jid.JID
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
	if err := dec.Decode(&r.numberOfOccupants); err != nil {
		return err
	}
	r.UserToOccupant = make(map[jid.JID]jid.JID)
	for i := 0; i < r.numberOfOccupants; i++ {
		userJID, err := jid.NewFromBytes(buf)
		if err != nil {
			return err
		}
		occJID, err := jid.NewFromBytes(buf)
		if err != nil {
			return err
		}
		r.UserToOccupant[*userJID] = *occJID
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
	if err := enc.Encode(&r.numberOfOccupants); err != nil {
		return err
	}
	for userJID, occJID := range r.UserToOccupant {
		if err := userJID.ToBytes(buf); err != nil {
			return err
		}
		if err := occJID.ToBytes(buf); err != nil {
			return err
		}
	}
	if err := enc.Encode(&r.Locked); err != nil {
		return err
	}
	return nil
}

func (r *Room) AddOccupant(o *Occupant) {
	r.UserToOccupant[*o.BareJID.ToBareJID()] = *o.OccupantJID
	r.numberOfOccupants++
}
