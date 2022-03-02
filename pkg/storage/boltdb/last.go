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

	lastmodel "github.com/ortuman/jackal/pkg/model/last"
	bolt "go.etcd.io/bbolt"
)

const lastKey = "lst"

type boltDBLastRep struct {
	tx *bolt.Tx
}

func newLastRep(tx *bolt.Tx) *boltDBLastRep {
	return &boltDBLastRep{tx: tx}
}

func (r *boltDBLastRep) UpsertLast(_ context.Context, last *lastmodel.Last) error {
	op := upsertKeyOp{
		tx:     r.tx,
		bucket: lastBucketKey(last.Username),
		key:    lastKey,
		obj:    last,
	}
	return op.do()
}

func (r *boltDBLastRep) FetchLast(_ context.Context, username string) (*lastmodel.Last, error) {
	op := fetchKeyOp{
		tx:     r.tx,
		bucket: lastBucketKey(username),
		key:    lastKey,
		obj:    &lastmodel.Last{},
	}
	obj, err := op.do()
	if err != nil {
		return nil, err
	}
	switch {
	case obj != nil:
		return obj.(*lastmodel.Last), nil
	default:
		return nil, nil
	}
}

func (r *boltDBLastRep) DeleteLast(_ context.Context, username string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: lastBucketKey(username),
	}
	return op.do()
}

func lastBucketKey(username string) string {
	return fmt.Sprintf("last:%s", username)
}

// UpsertLast satisfies repository.Last interface.
func (r *Repository) UpsertLast(ctx context.Context, last *lastmodel.Last) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newLastRep(tx).UpsertLast(ctx, last)
	})
}

// FetchLast satisfies repository.Last interface.
func (r *Repository) FetchLast(ctx context.Context, username string) (lst *lastmodel.Last, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		lst, err = newLastRep(tx).FetchLast(ctx, username)
		return err
	})
	return
}

// DeleteLast satisfies repository.Last interface.
func (r *Repository) DeleteLast(ctx context.Context, username string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newLastRep(tx).DeleteLast(ctx, username)
	})
}
