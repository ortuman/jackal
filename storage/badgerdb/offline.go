/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/xml"
)

func (b *Storage) InsertOfflineMessage(message xml.XElement, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(message, b.offlineMessageKey(username, message.ID()), tx)
	})
}

func (b *Storage) CountOfflineMessages(username string) (int, error) {
	cnt := 0
	prefix := []byte("offlineMessages:" + username)
	err := b.forEachKey(prefix, func(key []byte) error {
		cnt++
		return nil
	})
	return cnt, err
}

func (b *Storage) FetchOfflineMessages(username string) ([]xml.XElement, error) {
	var msgs []xml.Element
	if err := b.fetchAll(&msgs, []byte("offlineMessages:"+username)); err != nil {
		return nil, err
	}
	switch len(msgs) {
	case 0:
		return nil, nil
	default:
		ret := make([]xml.XElement, len(msgs))
		for i := 0; i < len(msgs); i++ {
			ret[i] = &msgs[i]
		}
		return ret, nil
	}
}

func (b *Storage) DeleteOfflineMessages(username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.deletePrefix([]byte("offlineMessages:"+username), tx)
	})
}

func (b *Storage) offlineMessageKey(username, identifier string) []byte {
	return []byte("offlineMessages:" + username + ":" + identifier)
}
