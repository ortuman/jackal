/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/serializer"
)

// InsertBlockListItems inserts a set of block list item entities
// into storage, only in case they haven't been previously inserted.
func (m *Storage) InsertBlockListItems(items []model.BlockListItem) error {
	return m.inWriteLock(func() error {
		for _, item := range items {
			blItems, err := m.fetchUserBlockListItems(item.Username)
			if err != nil {
				return err
			}
			if blItems != nil {
				for _, blItem := range blItems {
					if blItem.JID == item.JID {
						goto done
					}
				}
				blItems = append(blItems, item)
			} else {
				blItems = []model.BlockListItem{item}
			}
			if err := m.upsertBlockListItems(blItems, item.Username); err != nil {
				return err
			}
		done:
		}
		return nil
	})
}

// DeleteBlockListItems deletes a set of block list item entities from storage.
func (m *Storage) DeleteBlockListItems(items []model.BlockListItem) error {
	return m.inWriteLock(func() error {
		for _, itm := range items {
			blItems, err := m.fetchUserBlockListItems(itm.Username)
			if err != nil {
				return err
			}
			for i, blItem := range blItems {
				if blItem.JID == itm.JID {
					// delete item
					blItems = append(blItems[:i], blItems[i+1:]...)
					if err := m.upsertBlockListItems(blItems, itm.Username); err != nil {
						return err
					}
					break
				}
			}
		}
		return nil
	})
}

// FetchBlockListItems retrieves from storage all block list item entities
// associated to a given user.
func (m *Storage) FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	var blItems []model.BlockListItem
	if err := m.inReadLock(func() error {
		var fnErr error
		blItems, fnErr = m.fetchUserBlockListItems(username)
		return fnErr
	}); err != nil {
		return nil, err
	}
	return blItems, nil
}

func (m *Storage) upsertBlockListItems(blItems []model.BlockListItem, username string) error {
	b, err := serializer.SerializeSlice(&blItems)
	if err != nil {
		return err
	}
	m.bytes[blockListItemKey(username)] = b
	return nil
}

func (m *Storage) fetchUserBlockListItems(username string) ([]model.BlockListItem, error) {
	b := m.bytes[blockListItemKey(username)]
	if b == nil {
		return nil, nil
	}
	var blItems []model.BlockListItem
	if err := serializer.DeserializeSlice(b, &blItems); err != nil {
		return nil, err
	}
	return blItems, nil
}

func blockListItemKey(username string) string {
	return "blockListItems:" + username
}
