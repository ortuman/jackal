/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	mucmodel "github.com/ortuman/jackal/model/muc"
)

// Room represents an in-memory room storage.
type Room struct {
	*memoryStorage
}

// NewRoom returns an instance of Room in-memory storage.
func NewRoom() *Room {
	return &Room{memoryStorage: newStorage()}
}

// UpsertRoom inserts a new room entity into storage, or updates it in case it's been previously inserted.
func (m *Room) UpsertRoom(_ context.Context, room *mucmodel.Room) error {
	return m.saveEntity(roomKey(room.Name), room)
}

// DeleteRoom deletes a room entity from storage.
func (m *Room) DeleteRoom(_ context.Context, roomName string) error {
	return m.deleteKey(roomKey(roomName))
}

// FetchRoom retrieves from storage a room entity.
func (m *Room) FetchRoom(_ context.Context, roomName string) (*mucmodel.Room, error) {
	var room mucmodel.Room
	ok, err := m.getEntity(roomKey(roomName), &room)
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
func (m *Room) RoomExists(_ context.Context, roomName string) (bool, error) {
	return m.keyExists(roomKey(roomName))
}

func roomKey(roomName string) string {
	return "rooms:" + roomName
}
