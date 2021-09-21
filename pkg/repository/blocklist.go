// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repository

import (
	"context"

	blocklistmodel "github.com/ortuman/jackal/pkg/model/blocklist"
)

// BlockList defines storage operations for user's block list
type BlockList interface {
	// UpsertBlockListItem upserts a block list item entity into storage.
	UpsertBlockListItem(ctx context.Context, item *blocklistmodel.Item) error

	// DeleteBlockListItem deletes a block list item entity from storage.
	DeleteBlockListItem(ctx context.Context, item *blocklistmodel.Item) error

	// FetchBlockListItems retrieves from storage all block list items associated to a user.
	FetchBlockListItems(ctx context.Context, username string) ([]*blocklistmodel.Item, error)

	// DeleteBlockListItems deletes all block list items associated to a user.
	DeleteBlockListItems(ctx context.Context, username string) error
}
