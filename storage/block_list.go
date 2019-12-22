/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import "github.com/ortuman/jackal/model"

// blockListStorage defines storage operations for user's block list
type blockListStorage interface {
	InsertBlockListItem(item *model.BlockListItem) error
	DeleteBlockListItem(item *model.BlockListItem) error

	FetchBlockListItems(username string) ([]model.BlockListItem, error)
}

// InsertBlockListItem inserts a block list item entity into storage, only in case they haven't been previously inserted.
func InsertBlockListItem(item *model.BlockListItem) error {
	return instance().InsertBlockListItem(item)
}

// DeleteBlockListItem deletes a block list item entity from storage.
func DeleteBlockListItem(item *model.BlockListItem) error {
	return instance().DeleteBlockListItem(item)
}

// FetchBlockListItems retrieves from storage all block list item entities
// associated to a given user.
func FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	return instance().FetchBlockListItems(username)
}
