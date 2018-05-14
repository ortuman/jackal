/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/storage/model"
)

func (b *Storage) InsertOrUpdateUser(user *model.User) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.insertOrUpdate(user, b.userKey(user.Username), tx)
	})
}

func (b *Storage) DeleteUser(username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.userKey(username), tx)
	})
}

func (b *Storage) FetchUser(username string) (*model.User, error) {
	var usr model.User
	err := b.fetch(&usr, b.userKey(username))
	switch err {
	case nil:
		return &usr, nil
	case errBadgerDBEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *Storage) UserExists(username string) (bool, error) {
	err := b.fetch(nil, b.userKey(username))
	switch err {
	case nil:
		return true, nil
	case errBadgerDBEntityNotFound:
		return false, nil
	default:
		return false, err
	}
}

func (b *Storage) userKey(username string) []byte {
	return []byte("users:" + username)
}
