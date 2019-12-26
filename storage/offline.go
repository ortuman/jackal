/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"context"

	"github.com/ortuman/jackal/xmpp"
)

// offlineStorage defines storage operations for offline messages
type offlineStorage interface {
	InsertOfflineMessage(ctx context.Context, message *xmpp.Message, username string) error
	CountOfflineMessages(ctx context.Context, username string) (int, error)
	FetchOfflineMessages(ctx context.Context, username string) ([]xmpp.Message, error)
	DeleteOfflineMessages(ctx context.Context, username string) error
}

// InsertOfflineMessage inserts a new message element into user's offline queue.
func InsertOfflineMessage(ctx context.Context, message *xmpp.Message, username string) error {
	return instance().InsertOfflineMessage(ctx, message, username)
}

// CountOfflineMessages returns current length of user's offline queue.
func CountOfflineMessages(ctx context.Context, username string) (int, error) {
	return instance().CountOfflineMessages(ctx, username)
}

// FetchOfflineMessages retrieves from storage current user offline queue.
func FetchOfflineMessages(ctx context.Context, username string) ([]xmpp.Message, error) {
	return instance().FetchOfflineMessages(ctx, username)
}

// DeleteOfflineMessages clears a user offline queue.
func DeleteOfflineMessages(ctx context.Context, username string) error {
	return instance().DeleteOfflineMessages(ctx, username)
}
