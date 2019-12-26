/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"context"

	"github.com/ortuman/jackal/model"
)

// blockListStorage defines storage operations for user's block list
type blockListStorage interface {
	InsertBlockListItem(ctx context.Context, item *model.BlockListItem) error
	DeleteBlockListItem(ctx context.Context, item *model.BlockListItem) error

	FetchBlockListItems(ctx context.Context, username string) ([]model.BlockListItem, error)
}

// InsertBlockListItem inserts a block list item entity into storage, only in case they haven't been previously inserted.
func InsertBlockListItem(ctx context.Context, item *model.BlockListItem) error {
	return instance().InsertBlockListItem(ctx, item)
}

// DeleteBlockListItem deletes a block list item entity from storage.
func DeleteBlockListItem(ctx context.Context, item *model.BlockListItem) error {
	return instance().DeleteBlockListItem(ctx, item)
}

// FetchBlockListItems retrieves from storage all block list item entities
// associated to a given user.
func FetchBlockListItems(ctx context.Context, username string) ([]model.BlockListItem, error) {
	return instance().FetchBlockListItems(ctx, username)
}
