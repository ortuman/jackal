/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import "context"

// Container interface brings together all repository instances.
type Container interface {
	// User method returns repository.User concrete implementation.
	User() User

	// Roster method returns repository.Roster concrete implementation.
	Roster() Roster

	// Presences method returns repository.Presences concrete implementation.
	Presences() Presences

	// VCard method returns repository.VCard concrete implementation.
	VCard() VCard

	// Private method returns repository.Private concrete implementation.
	Private() Private

	// BlockList method returns repository.BlockList concrete implementation.
	BlockList() BlockList

	// PubSub method returns repository.PubSub concrete implementation.
	PubSub() PubSub

	// Offline method returns repository.Offline concrete implementation.
	Offline() Offline

	// Close closes underlying storage resources, commonly shared across repositories.
	Close(ctx context.Context) error

	// IsClusterCompatible tells whether or not container instance can be safely used across multiple cluster nodes.
	IsClusterCompatible() bool

	// Room method returns respository.Room concrete implementation
	Room() Room

	// Occupant method returns repository.Occupant concrete implementation
	Occupant() Occupant
}
