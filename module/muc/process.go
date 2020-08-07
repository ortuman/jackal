/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package muc

import (
	"context"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

func (s *Service) ProcessPresence(ctx context.Context, presence *xmpp.Presence) {
	xEl := presence.Elements().ChildNamespace("x", mucNamespace)
	// join a room if it exists, create it if it doesn't
	if xEl != nil && xEl.Text() == "" {
		from := presence.FromJID()
		to := presence.ToJID()
		// TODO this is a placeholder, user needs to enter a roomname
		roomName := "Chatroom"
		roomJID := to.ToBareJID()
		nick := to.Resource()
		exists, _ := s.reps.Room().RoomExists(ctx, roomJID)
		if exists {
			log.Infof("Room exists")
		} else {
			err := s.newRoom(ctx, from, to, roomJID, roomName, nick)
			if err != nil {
				log.Infof("New room created named %s", roomName)
			}
		}
	}
}

func (s *Service) newRoom(ctx context.Context, from, to, roomJID *jid.JID, roomName, ownerNick string) error {
	owner, err := s.CreateOwner(ctx, to, ownerNick, from)
	if err != nil {
		return err
	}
	room, err := s.CreateRoom(ctx, roomName, roomJID, owner)
	if err != nil {
		return err
	}
	s.publicRooms = append(s.publicRooms, room)
	return nil
}
