/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/serializer"
)

type User struct {
	*memoryStorage
}

func NewUser() *User {
	return &User{memoryStorage: newStorage()}
}

// UpsertUser inserts a new user entity into storage, or updates it in case it's been previously inserted.
func (m *User) UpsertUser(_ context.Context, user *model.User) error {
	b, err := serializer.Serialize(user)
	if err != nil {
		return err
	}
	return m.inWriteLock(func() error {
		m.b[userKey(user.Username)] = b
		return nil
	})
}

// DeleteUser deletes a user entity from storage.
func (m *User) DeleteUser(_ context.Context, username string) error {
	return m.inWriteLock(func() error {
		delete(m.b, userKey(username))
		return nil
	})
}

// FetchUser retrieves from storage a user entity.
func (m *User) FetchUser(_ context.Context, username string) (*model.User, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[userKey(username)]
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
func (m *User) UserExists(_ context.Context, username string) (bool, error) {
	var b []byte
	if err := m.inReadLock(func() error {
		b = m.b[userKey(username)]
		return nil
	}); err != nil {
		return false, err
	}
	return b != nil, nil
}

func userKey(username string) string {
	return "users:" + username
}
