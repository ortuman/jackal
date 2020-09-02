/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"fmt"

	"github.com/ortuman/jackal/log"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp"
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
	nicks := make(map[string]*mucmodel.Occupant)
	nicks[owner.FullJID.ToBareJID().String()] = owner

	r := &mucmodel.Room{
		Name:           name,
		RoomJID:        roomJID,
		Desc:           defaultDesc,
		Config:         getDefaultRoomConfig(),
		OccupantsCnt:   1,
		NickToOccupant: m,
		UserToOccupant:     nicks,
		Locked:         locked,
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

func (s *Muc) createInstantRoom(ctx context.Context, room *mucmodel.Room, iq *xmpp.IQ) {
	fromJID, err := jid.NewWithString(iq.From(), true)
	if err != nil {
		log.Error(err)
		_ = s.router.Route(ctx, iq.InternalServerError())
		return
	}
	occ, ok := room.UserToOccupant[fromJID.ToBareJID().String()]
	if !ok {
		_ = s.router.Route(ctx, iq.BadRequestError())
		return
	}
	if occ.Affiliation != "owner" {
		_ = s.router.Route(ctx, iq.NotAuthorizedError())
		return
	}

	room.Locked = false
	s.reps.Room().UpsertRoom(ctx, room)
	_ = s.router.Route(ctx, iq.ResultIQ())
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
