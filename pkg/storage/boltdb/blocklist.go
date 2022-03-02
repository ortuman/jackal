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

	blocklistmodel "github.com/ortuman/jackal/pkg/model/blocklist"
	bolt "go.etcd.io/bbolt"
)

type boltDBBlockListRep struct {
	tx *bolt.Tx
}

func newBlockListRep(tx *bolt.Tx) *boltDBBlockListRep {
	return &boltDBBlockListRep{tx: tx}
}

func (r *boltDBBlockListRep) UpsertBlockListItem(_ context.Context, item *blocklistmodel.Item) error {
	op := upsertKeyOp{
		tx:     r.tx,
		bucket: blockListBucket(item.Username),
		key:    item.Jid,
		obj:    item,
	}
	return op.do()
}

func (r *boltDBBlockListRep) DeleteBlockListItem(_ context.Context, item *blocklistmodel.Item) error {
	op := delKeyOp{
		tx:     r.tx,
		bucket: blockListBucket(item.Username),
		key:    item.Jid,
	}
	return op.do()
}

func (r *boltDBBlockListRep) FetchBlockListItems(_ context.Context, username string) ([]*blocklistmodel.Item, error) {
	var retVal []*blocklistmodel.Item

	op := iterKeysOp{
		tx:     r.tx,
		bucket: blockListBucket(username),
		iterFn: func(_, b []byte) error {
			var item blocklistmodel.Item
			if err := item.UnmarshalBinary(b); err != nil {
				return err
			}
			retVal = append(retVal, &item)
			return nil
		},
	}
	if err := op.do(); err != nil {
		return nil, err
	}
	return retVal, nil
}

func (r *boltDBBlockListRep) DeleteBlockListItems(_ context.Context, username string) error {
	op := delBucketOp{
		tx:     r.tx,
		bucket: blockListBucket(username),
	}
	return op.do()
}

func blockListBucket(username string) string {
	return fmt.Sprintf("blocklist:%s", username)
}

// UpsertBlockListItem satisfies repository.BlockList interface.
func (r *Repository) UpsertBlockListItem(ctx context.Context, item *blocklistmodel.Item) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newBlockListRep(tx).UpsertBlockListItem(ctx, item)
	})
}

// DeleteBlockListItem deletes a block list item entity from storage.
func (r *Repository) DeleteBlockListItem(ctx context.Context, item *blocklistmodel.Item) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newBlockListRep(tx).DeleteBlockListItem(ctx, item)
	})
}

// FetchBlockListItems retrieves from storage all block list items associated to a user.
func (r *Repository) FetchBlockListItems(ctx context.Context, username string) (items []*blocklistmodel.Item, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		items, err = newBlockListRep(tx).FetchBlockListItems(ctx, username)
		return err
	})
	return
}

// DeleteBlockListItems deletes all block list items associated to a user.
func (r *Repository) DeleteBlockListItems(ctx context.Context, username string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return newBlockListRep(tx).DeleteBlockListItems(ctx, username)
	})
}
