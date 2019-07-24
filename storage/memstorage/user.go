/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"github.com/ortuman/jackal/model"
)

// InsertOrUpdateUser inserts a new user entity into storage,
// or updates it in case it's been previously inserted.
func (m *Storage) InsertOrUpdateUser(user *model.User) error {
	return m.inWriteLock(func() error {
		m.users[user.Username] = user
		return nil
	})
}

// DeleteUser deletes a user entity from storage.
func (m *Storage) DeleteUser(username string) error {
	return m.inWriteLock(func() error {
		delete(m.users, username)
		return nil
	})
}

// FetchUser retrieves from storage a user entity.
func (m *Storage) FetchUser(username string) (*model.User, error) {
	var ret *model.User
	err := m.inReadLock(func() error {
		ret = m.users[username]
		return nil
	})
	return ret, err
}

// UserExists returns whether or not a user exists within storage.
func (m *Storage) UserExists(username string) (bool, error) {
	var ret bool
	err := m.inReadLock(func() error {
		ret = m.users[username] != nil
		return nil
	})
	return ret, err
}
