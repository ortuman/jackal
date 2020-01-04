/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package repository

import (
	"context"

	"github.com/ortuman/jackal/model"
)

// BlockList defines storage operations for user's block list
type BlockList interface {
	// InsertBlockListItem inserts a block list item entity into storage if not previously inserted.
	InsertBlockListItem(ctx context.Context, item *model.BlockListItem) error

	// DeleteBlockListItem deletes a block list item entity from storage.
	DeleteBlockListItem(ctx context.Context, item *model.BlockListItem) error

	// FetchBlockListItems retrieves from storage all block list item entities
	// associated to a given user.
	FetchBlockListItems(ctx context.Context, username string) ([]model.BlockListItem, error)
}
