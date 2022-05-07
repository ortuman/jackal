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

package cachedrepository

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/ortuman/jackal/pkg/model"
	usermodel "github.com/ortuman/jackal/pkg/model/user"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

const userKey = "usr"

type cachedUserRep struct {
	c      Cache
	rep    repository.User
	logger log.Logger
}

func (c *cachedUserRep) UpsertUser(ctx context.Context, user *usermodel.User) error {
	op := updateOp{
		c:              c.c,
		namespace:      userNS(user.Username),
		invalidateKeys: []string{userKey},
		updateFn: func(ctx context.Context) error {
			return c.rep.UpsertUser(ctx, user)
		},
	}
	return op.do(ctx)
}

func (c *cachedUserRep) DeleteUser(ctx context.Context, username string) error {
	op := updateOp{
		c:              c.c,
		namespace:      userNS(username),
		invalidateKeys: []string{userKey},
		updateFn: func(ctx context.Context) error {
			return c.rep.DeleteUser(ctx, username)
		},
	}
	return op.do(ctx)
}

func (c *cachedUserRep) FetchUser(ctx context.Context, username string) (*usermodel.User, error) {
	op := fetchOp{
		c:         c.c,
		namespace: userNS(username),
		key:       userKey,
		codec:     &usermodel.User{},
		missFn: func(ctx context.Context) (model.Codec, error) {
			return c.rep.FetchUser(ctx, username)
		},
		logger: c.logger,
	}
	v, err := op.do(ctx)
	switch {
	case err != nil:
		return nil, err
	case v != nil:
		return v.(*usermodel.User), nil
	}
	return nil, nil
}

func (c *cachedUserRep) UserExists(ctx context.Context, username string) (bool, error) {
	op := existsOp{
		c:         c.c,
		namespace: userNS(username),
		key:       userKey,
		missFn: func(ctx context.Context) (bool, error) {
			return c.rep.UserExists(ctx, username)
		},
		logger: c.logger,
	}
	return op.do(ctx)
}

func userNS(username string) string {
	return fmt.Sprintf("usr:%s", username)
}
