/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"fmt"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
)

const defaultDesc = "Chatroom"

func (s *Muc) newRoom(ctx context.Context, from, to *jid.JID, roomName, ownerNick string, locked bool) error {
	roomJID := to.ToBareJID()
	roomExists, _ := s.reps.Room().RoomExists(ctx, roomJID)
	// TODO this will probably be deleted since presence stanza to an existing room means join the
	// room
	if roomExists {
		return fmt.Errorf("Room %s already exists", roomName)
	}

	owner, err := s.createOwner(ctx, to, ownerNick, from)
	if err != nil {
		return err
	}
	room, err := s.createRoom(ctx, roomName, roomJID, owner, locked)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.allRooms = append(s.allRooms, room)
	s.mu.Unlock()
	return nil
}

func (s *Muc) createRoom(ctx context.Context, name string, roomJID *jid.JID, owner *mucmodel.Occupant, locked bool) (*mucmodel.Room, error) {
	m := make(map[string]*mucmodel.Occupant)
	m[owner.Nick] = owner
	r := &mucmodel.Room{
		Name:         name,
		RoomJID:      roomJID,
		Desc:         defaultDesc,
		Config:       getDefaultRoomConfig(),
		OccupantsCnt: 1,
		Occupants:    m,
		Locked:       locked,
	}
	err := s.reps.Room().UpsertRoom(ctx, r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Muc) sendRoomCreateAck(ctx context.Context, from, to *jid.JID) error {
	el := getAckStanza(from, to)
	err := s.router.Route(ctx, el)
	return err
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
