/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/model"
)

type badgerDBUser struct {
	*badgerDBStorage
}

func newUser(db *badger.DB) *badgerDBUser {
	return &badgerDBUser{badgerDBStorage: newStorage(db)}
}

func (b *badgerDBUser) UpsertUser(_ context.Context, user *model.User) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(user, b.userKey(user.Username), tx)
	})
}

func (b *badgerDBUser) DeleteUser(_ context.Context, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.userKey(username), tx)
	})
}

func (b *badgerDBUser) FetchUser(_ context.Context, username string) (*model.User, error) {
	var usr model.User
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(&usr, b.userKey(username), txn)
	})
	switch err {
	case nil:
		return &usr, nil
	case errEntityNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *badgerDBUser) UserExists(_ context.Context, username string) (bool, error) {
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetch(nil, b.userKey(username), txn)
	})
	switch err {
	case nil:
		return true, nil
	case errEntityNotFound:
		return false, nil
	default:
		return false, err
	}
}

func (b *badgerDBUser) userKey(username string) []byte {
	return []byte("users:" + username)
}
