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

package boltdb

import (
	"context"
	"fmt"

	usermodel "github.com/ortuman/jackal/pkg/model/user"
	bolt "go.etcd.io/bbolt"
)

const userKey = "usr"

type boltDBUserRep struct {
	tx *bolt.Tx
}

func newUserRep(tx *bolt.Tx) *boltDBUserRep {
	return &boltDBUserRep{tx: tx}
}

func (r *boltDBUserRep) UpsertUser(_ context.Context, user *usermodel.User) error {
	op := upsertKeyOp{
		tx:     r.tx,
		bucket: userBucketKey(user.Username),
		key:    userKey,
		obj:    user,
	}
	return op.do()
}

func (r *boltDBUserRep) DeleteUser(_ context.Context, username string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: userBucketKey(username),
	}
	return op.do()
}

func (r *boltDBUserRep) FetchUser(_ context.Context, username string) (*usermodel.User, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: userBucketKey(username),
		key:    userKey,
		obj:    &usermodel.User{},
	}
	obj, err := op.do()
	if err != nil {
		return nil, err
	}
	switch {
	case obj != nil:
		return obj.(*usermodel.User), nil
	default:
		return nil, nil
	}
}

func (r *boltDBUserRep) UserExists(_ context.Context, username string) (bool, error) {
	op := bucketExistsOp{
		tx:     r.tx,
		bucket: userBucketKey(username),
	}
	return op.do(), nil
}

func userBucketKey(username string) string {
	return fmt.Sprintf("user:%s", username)
}

// UpsertUser satisfies repository.User interface.
func (r *Repository) UpsertUser(ctx context.Context, user *usermodel.User) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newUserRep(tx).UpsertUser(ctx, user)
	})
}

// DeleteUser satisfies repository.User interface.
func (r *Repository) DeleteUser(ctx context.Context, username string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newUserRep(tx).DeleteUser(ctx, username)
	})
}

// FetchUser satisfies repository.User interface.
func (r *Repository) FetchUser(ctx context.Context, username string) (usr *usermodel.User, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		usr, err = newUserRep(tx).FetchUser(ctx, username)
		return err
	})
	return
}

// UserExists satisfies repository.User interface.
func (r *Repository) UserExists(ctx context.Context, username string) (ok bool, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		ok, err = newUserRep(tx).UserExists(ctx, username)
		return err
	})
	return
}
