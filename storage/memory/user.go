/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	"github.com/ortuman/jackal/model"
)

type User struct {
	*memoryStorage
}

func NewUser() *User {
	return &User{memoryStorage: newStorage()}
}

// UpsertUser inserts a new user entity into storage, or updates it in case it's been previously inserted.
func (m *User) UpsertUser(_ context.Context, user *model.User) error {
	return m.saveEntity(userKey(user.Username), user)
}

// DeleteUser deletes a user entity from storage.
func (m *User) DeleteUser(_ context.Context, username string) error {
	return m.deleteKey(userKey(username))
}

// FetchUser retrieves from storage a user entity.
func (m *User) FetchUser(_ context.Context, username string) (*model.User, error) {
	var user model.User
	ok, err := m.getEntity(userKey(username), &user)
	switch err {
	case nil:
		if ok {
			return &user, nil
		}
		return nil, nil
	default:
		return nil, err
	}
}

// UserExists returns whether or not a user exists within storage.
func (m *User) UserExists(_ context.Context, username string) (bool, error) {
	return m.keyExists(userKey(username))
}

func userKey(username string) string {
	return "users:" + username
}
