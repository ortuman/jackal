/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import "context"

type Container interface {
	User() User

	Capabilities() Capabilities

	VCard() VCard

	Private() Private

	BlockList() BlockList

	Offline() Offline

	Close(ctx context.Context) error

	IsClusterCompatible() bool
}
