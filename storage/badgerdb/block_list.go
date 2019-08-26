/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/model"
)

// InsertBlockListItems inserts a set of block list item entities
// into storage, only in case they haven't been previously inserted.
func (b *Storage) InsertBlockListItems(items []model.BlockListItem) error {
	return b.db.Update(func(tx *badger.Txn) error {
		for _, item := range items {
			if err := b.upsert(&item, b.blockListItemKey(item.Username, item.JID), tx); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteBlockListItems deletes a set of block list item entities from storage.
func (b *Storage) DeleteBlockListItems(items []model.BlockListItem) error {
	return b.db.Update(func(tx *badger.Txn) error {
		for _, item := range items {
			if err := b.delete(b.blockListItemKey(item.Username, item.JID), tx); err != nil {
				return err
			}
		}
		return nil
	})
}

// FetchBlockListItems retrieves from storage all block list item entities
// associated to a given user.
func (b *Storage) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	var blItems []model.BlockListItem
	err := b.db.View(func(txn *badger.Txn) error {
		return b.fetchAll(&blItems, []byte("blockListItems:"+username), txn)
	})
	if err != nil {
		return nil, err
	}
	return blItems, nil
}

func (b *Storage) blockListItemKey(username, jid string) []byte {
	return []byte("blockListItems:" + username + ":" + jid)
}
