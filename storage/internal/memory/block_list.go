/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memory

import (
	"context"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/serializer"
)

// InsertBlockListItem a block list item entity
// into storage, only in case they haven't been previously inserted.
func (m *Storage) InsertBlockListItem(_ context.Context, item *model.BlockListItem) error {
	return m.inWriteLock(func() error {
		blItems, err := m.fetchUserBlockListItems(item.Username)
		if err != nil {
			return err
		}
		if blItems != nil {
			for _, blItem := range blItems {
				if blItem.JID == item.JID {
					return nil
				}
			}
			blItems = append(blItems, *item)
		} else {
			blItems = []model.BlockListItem{*item}
		}
		if err := m.upsertBlockListItems(blItems, item.Username); err != nil {
			return err
		}
		return nil
	})
}

// DeleteBlockListItem deletes a of block list item entity from storage.
func (m *Storage) DeleteBlockListItem(_ context.Context, item *model.BlockListItem) error {
	return m.inWriteLock(func() error {
		blItems, err := m.fetchUserBlockListItems(item.Username)
		if err != nil {
			return err
		}
		for i, blItem := range blItems {
			if blItem.JID == item.JID {
				// delete item
				blItems = append(blItems[:i], blItems[i+1:]...)
				if err := m.upsertBlockListItems(blItems, item.Username); err != nil {
					return err
				}
				break
			}
		}
		return nil
	})
}

// FetchBlockListItems retrieves from storage all block list item entities
// associated to a given user.
func (m *Storage) FetchBlockListItems(_ context.Context, username string) ([]model.BlockListItem, error) {
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
