/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"

	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/model"
)

// InsertBlockListItem inserts a block list item entity
// into storage, only in case they haven't been previously inserted.
func (b *Storage) InsertBlockListItem(_ context.Context, item *model.BlockListItem) error {
	return b.db.Update(func(tx *badger.Txn) error {
		var blItems []model.BlockListItem
		if err := b.fetchSlice(&blItems, b.blockListItemsKey(item.Username), tx); err != nil {
			return err
		}
		for _, blItem := range blItems {
			if blItem.JID == item.JID {
				return nil
			}
		}
		blItems = append(blItems, *item)
		return b.upsertSlice(&blItems, b.blockListItemsKey(item.Username), tx)
	})
}

// DeleteBlockListItem deletes a block list item entity from storage.
func (b *Storage) DeleteBlockListItem(_ context.Context, item *model.BlockListItem) error {
	return b.db.Update(func(tx *badger.Txn) error {
		var blItems []model.BlockListItem
		if err := b.fetchSlice(&blItems, b.blockListItemsKey(item.Username), tx); err != nil {
			return err
		}
		for i, blItem := range blItems {
			if blItem.JID == item.JID { // delete item
				blItems = append(blItems[:i], blItems[i+1:]...)
				return b.upsertSlice(&blItems, b.blockListItemsKey(item.Username), tx)
			}
		}
		return nil
	})
}

// FetchBlockListItems retrieves from storage all block list item entities
// associated to a given user.
func (b *Storage) FetchBlockListItems(_ context.Context, username string) ([]model.BlockListItem, error) {
	var blItems []model.BlockListItem
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchSlice(&blItems, b.blockListItemsKey(username), txn)
	})
	if err != nil {
		return nil, err
	}
	return blItems, nil
}

func (b *Storage) blockListItemsKey(username string) []byte {
	return []byte("blockListItems:" + username)
}
