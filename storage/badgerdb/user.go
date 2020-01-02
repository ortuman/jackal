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

type User struct {
	*badgerDBStorage
}

func NewUser(db *badger.DB) *User {
	return &User{badgerDBStorage: newStorage(db)}
}

// UpsertUser inserts a new user entity into storage, or updates it in case it's been previously inserted.
func (b *User) UpsertUser(_ context.Context, user *model.User) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.upsert(user, b.userKey(user.Username), tx)
	})
}

// DeleteUser deletes a user entity from storage.
func (b *User) DeleteUser(_ context.Context, username string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return b.delete(b.userKey(username), tx)
	})
}

// FetchUser retrieves from storage a user entity.
func (b *User) FetchUser(_ context.Context, username string) (*model.User, error) {
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

// UserExists returns whether or not a user exists within storage.
func (b *User) UserExists(_ context.Context, username string) (bool, error) {
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

func (b *User) userKey(username string) []byte {
	return []byte("users:" + username)
}
