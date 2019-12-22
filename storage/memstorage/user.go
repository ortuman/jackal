/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/serializer"
)

// UpsertUser inserts a new user entity into storage,
// or updates it in case it's been previously inserted.
func (m *Storage) UpsertUser(user *model.User) error {
	b, err := serializer.Serialize(user)
	if err != nil {
		return err
	}
	return m.inWriteLock(func() error {
		m.bytes[userKey(user.Username)] = b
		return nil
	})
}

// DeleteUser deletes a user entity from storage.
func (m *Storage) DeleteUser(username string) error {
	return m.inWriteLock(func() error {
		delete(m.bytes, userKey(username))
		return nil
	})
}

// FetchUser retrieves from storage a user entity.
func (m *Storage) FetchUser(username string) (*model.User, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[userKey(username)]
		return nil
	}); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	var usr model.User
	if err := serializer.Deserialize(b, &usr); err != nil {
		return nil, err
	}
	return &usr, nil
}

// UserExists returns whether or not a user exists within storage.
func (m *Storage) UserExists(username string) (bool, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.bytes[userKey(username)]
		return nil
	}); err != nil {
		return false, err
	}
	return b != nil, nil
}

func userKey(username string) string {
	return "users:" + username
}
