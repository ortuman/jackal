/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"

	"github.com/ortuman/jackal/model"
)

// User defines user repository operations
type User interface {
	UpsertUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, username string) error
	FetchUser(ctx context.Context, username string) (*model.User, error)
	UserExists(ctx context.Context, username string) (bool, error)
}
