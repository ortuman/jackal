// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repository

import (
	"context"

	usermodel "github.com/ortuman/jackal/pkg/model/user"
)

// User defines user repository operations
type User interface {
	// UpsertUser inserts a new user entity into repository.
	UpsertUser(ctx context.Context, user *usermodel.User) error

	// DeleteUser deletes a user entity from repository.
	DeleteUser(ctx context.Context, username string) error

	// FetchUser retrieves a user entity from repository.
	FetchUser(ctx context.Context, username string) (*usermodel.User, error)

	// UserExists tells whether or not a user exists within repository.
	UserExists(ctx context.Context, username string) (bool, error)
}
