/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
)

// Room represents an in-memory room storage.
type Room struct {
	*memoryStorage
}

// NewRoom returns an instance of Room in-memory storage.
func NewRoom() *Room {
	return &Room{memoryStorage: newStorage()}
}

// UpsertRoom inserts a new room entity into storage, or updates the existing room.
func (m *Room) UpsertRoom(_ context.Context, room *mucmodel.Room) error {
	return m.saveEntity(roomKey(room.RoomJID), room)
}

// DeleteRoom deletes a room entity from storage.
func (m *Room) DeleteRoom(_ context.Context, roomJID *jid.JID) error {
	return m.deleteKey(roomKey(roomJID))
}

// FetchRoom retrieves from storage a room entity.
func (m *Room) FetchRoom(_ context.Context, roomJID *jid.JID) (*mucmodel.Room, error) {
	var room mucmodel.Room
	ok, err := m.getEntity(roomKey(roomJID), &room)
	switch err {
	case nil:
		if ok {
			return &room, nil
		}
		return nil, nil
	default:
		return nil, err
	}
}

// RoomExists returns whether or not a room exists within storage.
func (m *Room) RoomExists(_ context.Context, roomJID *jid.JID) (bool, error) {
	return m.keyExists(roomKey(roomJID))
}

func roomKey(roomJID *jid.JID) string {
	return "rooms:" + roomJID.String()
}
