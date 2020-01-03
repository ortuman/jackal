/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"

	"github.com/ortuman/jackal/model"
)

// User defines user repository operations
type User interface {
	// UpsertUser inserts a new user entity into storage, or updates it in case it's been previously inserted.
	UpsertUser(ctx context.Context, user *model.User) error

	// DeleteUser deletes a user entity from storage.
	DeleteUser(ctx context.Context, username string) error

	// FetchUser retrieves from storage a user entity.
	FetchUser(ctx context.Context, username string) (*model.User, error)

	// UserExists tells whether or not a user exists within storage.
	UserExists(ctx context.Context, username string) (bool, error)
}
