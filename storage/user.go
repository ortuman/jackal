/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"context"

	"github.com/ortuman/jackal/model"
)

// userStorage defines storage operations for users
type userStorage interface {
	UpsertUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, username string) error
	FetchUser(ctx context.Context, username string) (*model.User, error)
	UserExists(ctx context.Context, username string) (bool, error)
}

// UpsertUser inserts a new user entity into storage, or updates it in case it's been previously inserted.
func UpsertUser(ctx context.Context, user *model.User) error {
	return instance().UpsertUser(ctx, user)
}

// DeleteUser deletes a user entity from storage.
func DeleteUser(ctx context.Context, username string) error {
	return instance().DeleteUser(ctx, username)
}

// FetchUser retrieves from storage a user entity.
func FetchUser(ctx context.Context, username string) (*model.User, error) {
	return instance().FetchUser(ctx, username)
}

// UserExists returns whether or not a user exists within storage.
func UserExists(ctx context.Context, username string) (bool, error) {
	return instance().UserExists(ctx, username)
}
