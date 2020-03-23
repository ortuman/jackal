/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package muc

import (
	"context"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
)

const moderatorRole = "moderator"
const ownerAff = "owner"
const defaultDesc = "Chatroom"

func (s *Service) CreateOwner(ctx context.Context, occJID *jid.JID, nick string, fullJID *jid.JID) (*mucmodel.Occupant, error) {
	o := &mucmodel.Occupant{
		OccupantJID: occJID,
		Nick:        nick,
		FullJID:     fullJID,
		Affiliation: ownerAff,
		Role:        moderatorRole,
	}
	err := s.reps.Occupant().UpsertOccupant(ctx, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *Service) CreateRoom(ctx context.Context, name string, roomJID *jid.JID, owner *mucmodel.Occupant) (*mucmodel.Room, error) {
	m := make(map[string]*mucmodel.Occupant)
	m[owner.Nick] = owner
	r := &mucmodel.Room{
		Name:         name,
		RoomJID:      roomJID,
		Desc:         defaultDesc,
		Config:       getDefaultRoomConfig(),
		OccupantsCnt: 1,
		Occupants:    m,
		Locked:       true,
	}
	err := s.reps.Room().UpsertRoom(ctx, r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func getDefaultRoomConfig() *mucmodel.RoomConfig {
	return &mucmodel.RoomConfig{
		Public:       true,
		Persistent:   true,
		PwdProtected: false,
		Open:         true,
		Moderated:    false,
		NonAnonymous: true,
	}
}
