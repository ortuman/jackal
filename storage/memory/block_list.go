/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/serializer"
)

// BlockList represents an in-memory block-list storage.
type BlockList struct {
	*memoryStorage
}

// NewBlockList returns an instance of BlockList in-memory storage.
func NewBlockList() *BlockList {
	return &BlockList{memoryStorage: newStorage()}
}

// InsertBlockListItem inserts a block list item entity into storage if not previously inserted.
func (m *BlockList) InsertBlockListItem(_ context.Context, item *model.BlockListItem) error {
	return m.updateInWriteLock(blockListItemKey(item.Username), func(b []byte) ([]byte, error) {
		var items []model.BlockListItem
		if len(b) > 0 {
			if err := serializer.DeserializeSlice(b, &items); err != nil {
				return nil, err
			}
		}
		for _, itm := range items {
			if itm.JID == item.JID {
				return b, nil // already inserted
			}
		}
		items = append(items, *item)

		output, err := serializer.SerializeSlice(&items)
		if err != nil {
			return nil, err
		}
		return output, nil
	})
}

// DeleteBlockListItem deletes a block list item entity from storage.
func (m *BlockList) DeleteBlockListItem(_ context.Context, item *model.BlockListItem) error {
	return m.updateInWriteLock(blockListItemKey(item.Username), func(b []byte) ([]byte, error) {
		var items []model.BlockListItem
		if len(b) > 0 {
			if err := serializer.DeserializeSlice(b, &items); err != nil {
				return nil, err
			}
		}
		for i, itm := range items {
			if itm.JID == item.JID {
				items = append(items[:i], items[i+1:]...)

				output, err := serializer.SerializeSlice(&items)
				if err != nil {
					return nil, err
				}
				return output, nil
			}
		}
		return b, nil // not present
	})
}

// FetchBlockListItems retrieves from storage all block list item entities associated to a given user.
func (m *BlockList) FetchBlockListItems(_ context.Context, username string) ([]model.BlockListItem, error) {
	var items []model.BlockListItem
	_, err := m.getEntities(blockListItemKey(username), &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func blockListItemKey(username string) string {
	return "blockListItems:" + username
}
