/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
)

// Offline defines storage operations for offline messages
type Offline interface {
	// InsertOfflineMessage inserts a new message element into user's offline queue.
	InsertOfflineMessage(ctx context.Context, message *xmpp.Message, username string) error

	// CountOfflineMessages returns current length of user's offline queue.
	CountOfflineMessages(ctx context.Context, username string) (int, error)

	// FetchOfflineMessages retrieves from storage current user offline queue.
	FetchOfflineMessages(ctx context.Context, username string) ([]xmpp.Message, error)

	// DeleteOfflineMessages clears a user offline queue.
	DeleteOfflineMessages(ctx context.Context, username string) error
}
