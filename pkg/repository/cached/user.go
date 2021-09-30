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

	"github.com/golang/protobuf/proto"
	usermodel "github.com/ortuman/jackal/pkg/model/user"
	"github.com/ortuman/jackal/pkg/repository"
)

const keyPrefix = "usr:"

type cachedUserRepository struct {
	c       Cache
	baseRep repository.User
}

func (c *cachedUserRepository) UpsertUser(ctx context.Context, user *usermodel.User) error {
	op := writeOp{
		c:   c.c,
		key: getUserKey(user.Username),
		fn: func(ctx context.Context) error {
			return c.baseRep.UpsertUser(ctx, user)
		},
	}
	return op.perform(ctx)
}

func (c *cachedUserRepository) DeleteUser(ctx context.Context, username string) error {
	op := &writeOp{
		c:   c.c,
		key: getUserKey(username),
		fn: func(ctx context.Context) error {
			return c.baseRep.DeleteUser(ctx, username)
		},
	}
	return op.perform(ctx)
}

func (c *cachedUserRepository) FetchUser(ctx context.Context, username string) (*usermodel.User, error) {
	var usr usermodel.User

	op := &readOp{
		c:   c.c,
		key: getUserKey(username),
		fn: func(ctx context.Context) (proto.Message, error) {
			return c.baseRep.FetchUser(ctx, username)
		},
		obj: &usr,
	}
	if err := op.perform(ctx); err != nil {
		return nil, err
	}
	if !op.fetched {
		return nil, nil
	}
	return op.obj.(*usermodel.User), nil
}

func (c *cachedUserRepository) UserExists(ctx context.Context, username string) (bool, error) {
	exists, err := c.c.Exists(ctx, getUserKey(username))
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	return c.baseRep.UserExists(ctx, username)
}

func getUserKey(k string) string {
	return fmt.Sprintf("%s:%s", keyPrefix, k)
}
