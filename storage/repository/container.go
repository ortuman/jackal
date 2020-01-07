/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import "context"

type Container interface {
	User() User

	Roster() Roster

	Capabilities() Capabilities

	VCard() VCard

	Private() Private

	BlockList() BlockList

	PubSub() PubSub

	Offline() Offline

	Close(ctx context.Context) error

	IsClusterCompatible() bool
}
