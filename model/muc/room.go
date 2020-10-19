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
	"github.com/ortuman/jackal/log"
)

type Room struct {
	Config            *RoomConfig
	RoomJID           *jid.JID
	Name              string
	Desc              string
	Subject           string
	Language          string
	Locked            bool
	//mapping user bare jid to the occupant JID
	userToOccupant map[jid.JID]jid.JID
	// a set of invited users' bare JIDs who haven't accepted the invitation yet
	invitedUsers    map[jid.JID]bool
	occupantsOnline int
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
	var numberOfOccupants int
	if err := dec.Decode(&numberOfOccupants); err != nil {
		return err
	}
	r.userToOccupant = make(map[jid.JID]jid.JID)
	for i := 0; i < numberOfOccupants; i++ {
		userJID, err := jid.NewFromBytes(buf)
		if err != nil {
			return err
		}
		occJID, err := jid.NewFromBytes(buf)
		if err != nil {
			return err
		}
		r.userToOccupant[*userJID] = *occJID
	}
	if err := dec.Decode(&r.Locked); err != nil {
		return err
	}
	var invitedUsersCount int
	if err := dec.Decode(&invitedUsersCount); err != nil {
		return err
	}
	r.invitedUsers = make(map[jid.JID]bool)
	for i := 0; i < invitedUsersCount; i++ {
		userJID, err := jid.NewFromBytes(buf)
		if err != nil {
			return err
		}
		r.invitedUsers[*userJID] = true
	}
	if err := dec.Decode(&r.occupantsOnline); err != nil {
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
	if err := enc.Encode(len(r.userToOccupant)); err != nil {
		return err
	}
	for userJID, occJID := range r.userToOccupant {
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
	if err := enc.Encode(len(r.invitedUsers)); err != nil {
		return err
	}
	for userJID, _ := range r.invitedUsers {
		if err := userJID.ToBytes(buf); err != nil {
			return err
		}
	}
	if err := enc.Encode(&r.occupantsOnline); err != nil {
		return err
	}
	return nil
}

func (r *Room) AddOccupant(o *Occupant) {
	// if this user was invited, remove from the list of invited users
	if r.UserIsInvited(o.BareJID.ToBareJID()) {
		o.SetAffiliation("member")
		r.DeleteInvite(o.BareJID.ToBareJID())
	}

	err := r.mapUserToOccupantJID(o.BareJID, o.OccupantJID)
	if err != nil {
		log.Error(err)
		return
	}

	if o.HasNoRole() {
		r.SetDefaultRole(o)
	}

	r.occupantsOnline++
}

func (r *Room) RemoveOccupant(o *Occupant) {
	delete(r.userToOccupant, *o.BareJID)
	r.occupantsOnline--
}

func (r *Room) SetDefaultRole(o *Occupant) {
	if o.IsOwner() || o.IsAdmin() {
		o.SetRole(moderator)
	} else if r.Config.Moderated && o.GetAffiliation() == "" {
		o.SetRole(visitor)
	} else {
		o.SetRole(participant)
	}
}

func (r *Room) mapUserToOccupantJID(userJID, occJID *jid.JID) error {
	if !occJID.IsFullWithUser() {
		return fmt.Errorf("Occupant JID %s is not valid", occJID.String())
	}
	if !userJID.IsBare() {
		return fmt.Errorf("User JID %s is not a bare JID", userJID.String())
	}

	if r.userToOccupant == nil {
		r.userToOccupant = make(map[jid.JID]jid.JID)
	}

	_, found := r.userToOccupant[*userJID]
	if !found {
		r.userToOccupant[*userJID] = *occJID
	}

	return nil
}

func (r *Room) GetOccupantJID(userJID *jid.JID) (jid.JID, bool) {
	occJID, found := r.userToOccupant[*userJID]
	return occJID, found
}

func (r *Room) GetAllOccupantJIDs() []jid.JID {
	res := make([]jid.JID, 0, len(r.userToOccupant))
	for _, occJID := range r.userToOccupant {
		res = append(res, occJID)
	}
	return res
}

func (r *Room) UserIsInRoom(userJID *jid.JID) bool {
	_, found := r.userToOccupant[*userJID]
	return found
}

func (r *Room) InviteUser(userJID *jid.JID) error {
	if r.invitedUsers == nil {
		r.invitedUsers = make(map[jid.JID]bool)
	}

	if !userJID.IsBare() {
		return fmt.Errorf("User JID %s is not a bare JID", userJID)
	}

	r.invitedUsers[*userJID] = true
	return nil
}

func (r *Room) UserIsInvited(userJID *jid.JID) bool {
	if r.invitedUsers == nil {
		return false
	}

	_, invited := r.invitedUsers[*userJID]
	return invited
}

func (r *Room) DeleteInvite(userJID *jid.JID) {
	delete(r.invitedUsers, *userJID)
}

func (r *Room) Full() bool {
	return r.occupantsOnline >= r.Config.MaxOccCnt
}
