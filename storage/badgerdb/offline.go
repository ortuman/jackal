/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/xmpp"
)

// InsertOfflineMessage inserts a new message element into
// user's offline queue.
func (b *Storage) InsertOfflineMessage(message *xmpp.Message, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(message, b.offlineMessageKey(username, message.ID()), tx)
	})
}

// CountOfflineMessages returns current length of user's offline queue.
func (b *Storage) CountOfflineMessages(username string) (int, error) {
	cnt := 0
	prefix := []byte("offlineMessages:" + username)
	err := b.forEachKey(prefix, func(key []byte) error {
		cnt++
		return nil
	})
	return cnt, err
}

// FetchOfflineMessages retrieves from storage current user offline queue.
func (b *Storage) FetchOfflineMessages(username string) ([]xmpp.Message, error) {
	var msgs []xmpp.Message
	if err := b.fetchAll(&msgs, []byte("offlineMessages:"+username)); err != nil {
		return nil, err
	}
	switch len(msgs) {
	case 0:
		return nil, nil
	default:
		ret := make([]xmpp.Message, len(msgs))
		for i := 0; i < len(msgs); i++ {
			ret[i] = msgs[i]
		}
		return ret, nil
	}
}

// DeleteOfflineMessages clears a user offline queue.
func (b *Storage) DeleteOfflineMessages(username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.deletePrefix([]byte("offlineMessages:"+username), tx)
	})
}

func (b *Storage) offlineMessageKey(username, identifier string) []byte {
	return []byte("offlineMessages:" + username + ":" + identifier)
}
