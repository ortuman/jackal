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

package measuredrepository

import (
	"context"
	"testing"

	blocklistmodel "github.com/ortuman/jackal/model/blocklist"
	"github.com/stretchr/testify/require"
)

func TestMeasuredBlockListRep_UpsertBlockListItem(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertBlockListItemFunc = func(ctx context.Context, item *blocklistmodel.Item) error {
		return nil
	}
	m := New(repMock)

	// when
	_ = m.UpsertBlockListItem(context.Background(), &blocklistmodel.Item{})

	// then
	require.Len(t, repMock.UpsertBlockListItemCalls(), 1)
}

func TestMeasuredBlockListRep_FetchBlockListItems(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchBlockListItemsFunc = func(ctx context.Context, username string) ([]blocklistmodel.Item, error) {
		return []blocklistmodel.Item{}, nil
	}
	m := New(repMock)

	// when
	_, _ = m.FetchBlockListItems(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.FetchBlockListItemsCalls(), 1)
}

func TestMeasuredBlockListRep_DeleteBlockListItem(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteBlockListItemFunc = func(ctx context.Context, item *blocklistmodel.Item) error {
		return nil
	}
	m := New(repMock)

	// when
	_ = m.DeleteBlockListItem(context.Background(), &blocklistmodel.Item{})

	// then
	require.Len(t, repMock.DeleteBlockListItemCalls(), 1)
}

func TestMeasuredBlockListRep_DeleteBlockListItems(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteBlockListItemsFunc = func(ctx context.Context, username string) error {
		return nil
	}
	m := New(repMock)

	// when
	_ = m.DeleteBlockListItems(context.Background(), "usr-1")

	// then
	require.Len(t, repMock.DeleteBlockListItemsCalls(), 1)
}
