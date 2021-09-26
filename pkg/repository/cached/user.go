// Copyright 2021 The jackal Authors
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

package cachedrepository

import (
	"context"
	"fmt"

	usermodel "github.com/ortuman/jackal/pkg/model/user"
	"github.com/ortuman/jackal/pkg/repository"
)

const keyPrefix = "usr:"

type cachedUserRepository struct {
	c       Cache
	baseRep repository.User
}

func (c *cachedUserRepository) UpsertUser(ctx context.Context, user *usermodel.User) error {
	return nil
}

func (c *cachedUserRepository) DeleteUser(ctx context.Context, username string) error {
	return nil
}

func (c *cachedUserRepository) FetchUser(ctx context.Context, username string) (*usermodel.User, error) {
	return nil, nil
}

func (c *cachedUserRepository) UserExists(ctx context.Context, username string) (bool, error) {
	return false, nil
}

func getKey(k string) string {
	return fmt.Sprintf("%s:%s", keyPrefix, k)
}
