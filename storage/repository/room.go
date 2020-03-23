/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"
	mucmodel "github.com/ortuman/jackal/model/muc"
)

// Room defines room repository operations
type Room interface {
	// UpsertRoom inserts a new room entity into storage, or updates it if previously inserted.
	UpsertRoom(ctx context.Context, room *mucmodel.Room) error

	// DeleteRoom deletes a room entity from storage.
	DeleteRoom(ctx context.Context, roomName string) error

	// FetchRoom retrieves a room entity from storage.
	FetchRoom(ctx context.Context, roomName string) (*mucmodel.Room, error)

	// RoomExists tells whether or not a room exists within storage.
	RoomExists(ctx context.Context, roomName string) (bool, error)
}
