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
	if err := c.baseRep.UpsertUser(ctx, user); err != nil {
		return err
	}
	return c.c.Del(ctx, getKey(user.Username))
}

func (c *cachedUserRepository) DeleteUser(ctx context.Context, username string) error {
	if err := c.baseRep.DeleteUser(ctx, username); err != nil {
		return err
	}
	return c.c.Del(ctx, getKey(username))
}

func (c *cachedUserRepository) FetchUser(ctx context.Context, username string) (usr *usermodel.User, err error) {
	usr, err = c.fetchUser(ctx, username)
	if err != nil {
		return nil, err
	}
	if usr != nil {
		return usr, nil
	}
	usr, err = c.baseRep.FetchUser(ctx, username)
	if err != nil {
		return nil, err
	}
	if err := c.storeUser(ctx, usr); err != nil {
		return nil, err
	}
	return usr, err
}

func (c *cachedUserRepository) UserExists(ctx context.Context, username string) (bool, error) {
	exists, err := c.c.Exists(ctx, getKey(username))
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	return c.baseRep.UserExists(ctx, username)
}

func (c *cachedUserRepository) fetchUser(ctx context.Context, username string) (*usermodel.User, error) {
	b, err := c.c.Fetch(ctx, getKey(username))
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, nil
	}
	var usr usermodel.User
	if err := proto.Unmarshal(b, &usr); err != nil {
		return nil, err
	}
	return &usr, nil
}

func (c *cachedUserRepository) storeUser(ctx context.Context, usr *usermodel.User) error {
	b, err := proto.Marshal(usr)
	if err != nil {
		return err
	}
	return c.c.Store(ctx, getKey(usr.Username), b)
}

func getKey(k string) string {
	return fmt.Sprintf("%s:%s", keyPrefix, k)
}
