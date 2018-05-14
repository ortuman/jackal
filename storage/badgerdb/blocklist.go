/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"github.com/dgraph-io/badger"
	"github.com/ortuman/jackal/storage/model"
)

func (b *Storage) InsertOrUpdateBlockListItems(items []model.BlockListItem) error {
	return b.db.Update(func(tx *badger.Txn) error {
		for _, item := range items {
			if err := b.insertOrUpdate(&item, b.blockListItemKey(item.Username, item.JID), tx); err != nil {
				return err
			}
		}
		return nil
	})
}

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

func (b *Storage) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	var blItems []model.BlockListItem
	if err := b.fetchAll(&blItems, []byte("blockListItems:"+username)); err != nil {
		return nil, err
	}
	return blItems, nil
}

func (b *Storage) blockListItemKey(username, jid string) []byte {
	return []byte("blockListItems:" + username + ":" + jid)
}
