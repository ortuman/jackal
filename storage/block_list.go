package storage

import "github.com/ortuman/jackal/model"

// blockListStorage defines storage operations for user's block list
type blockListStorage interface {
	InsertBlockListItems(items []model.BlockListItem) error
	DeleteBlockListItems(items []model.BlockListItem) error
	FetchBlockListItems(username string) ([]model.BlockListItem, error)
}

// InsertBlockListItems inserts a set of block list item entities
// into storage, only in case they haven't been previously inserted.
func InsertBlockListItems(items []model.BlockListItem) error {
	return instance().InsertBlockListItems(items)
}

// DeleteBlockListItems deletes a set of block list item entities from storage.
func DeleteBlockListItems(items []model.BlockListItem) error {
	return instance().DeleteBlockListItems(items)
}

// FetchBlockListItems retrieves from storage all block list item entities
// associated to a given user.
func FetchBlockListItems(username string) ([]model.BlockListItem, error) {
	return instance().FetchBlockListItems(username)
}
