/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/xmpp"
)

// InsertOfflineMessage inserts a new message element into user's offline queue.
func (b *Storage) InsertOfflineMessage(_ context.Context, message *xmpp.Message, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		var messages []xmpp.Message
		if err := b.fetchSlice(&messages, b.offlineMessagesKey(username), tx); err != nil {
			return err
		}
		messages = append(messages, *message)
		return b.upsertSlice(&messages, b.offlineMessagesKey(username), tx)
	})
}

// CountOfflineMessages returns current length of user's offline queue.
func (b *Storage) CountOfflineMessages(_ context.Context, username string) (int, error) {
	var cnt int
	err := b.db.View(func(tx *badger.Txn) error {
		var messages []xmpp.Message
		if err := b.fetchSlice(&messages, b.offlineMessagesKey(username), tx); err != nil {
			return err
		}
		cnt = len(messages)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

// FetchOfflineMessages retrieves from storage current user offline queue.
func (b *Storage) FetchOfflineMessages(_ context.Context, username string) ([]xmpp.Message, error) {
	var messages []xmpp.Message
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchSlice(&messages, b.offlineMessagesKey(username), txn)
	})
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// DeleteOfflineMessages clears a user offline queue.
func (b *Storage) DeleteOfflineMessages(_ context.Context, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.offlineMessagesKey(username), tx)
	})
}

func (b *Storage) offlineMessagesKey(username string) []byte {
	return []byte("offlineMessages:" + username)
}
