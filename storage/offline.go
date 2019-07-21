/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import "github.com/ortuman/jackal/xmpp"

// offlineStorage defines storage operations for offline messages
type offlineStorage interface {
	InsertOfflineMessage(message *xmpp.Message, username string) error
	CountOfflineMessages(username string) (int, error)
	FetchOfflineMessages(username string) ([]xmpp.Message, error)
	DeleteOfflineMessages(username string) error
}

// InsertOfflineMessage inserts a new message element into
// user's offline queue.
func InsertOfflineMessage(message *xmpp.Message, username string) error {
	return instance().InsertOfflineMessage(message, username)
}

// CountOfflineMessages returns current length of user's offline queue.
func CountOfflineMessages(username string) (int, error) {
	return instance().CountOfflineMessages(username)
}

// FetchOfflineMessages retrieves from storage current user offline queue.
func FetchOfflineMessages(username string) ([]xmpp.Message, error) {
	return instance().FetchOfflineMessages(username)
}

// DeleteOfflineMessages clears a user offline queue.
func DeleteOfflineMessages(username string) error {
	return instance().DeleteOfflineMessages(username)
}
