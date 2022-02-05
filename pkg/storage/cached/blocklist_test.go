// Copyright 2022 The jackal Authors
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

package cachedrepository

import (
	"context"
	"testing"

	blocklistmodel "github.com/ortuman/jackal/pkg/model/blocklist"
	"github.com/stretchr/testify/require"
)

func TestCachedBlockListRep_UpsertBlockListItem(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertBlockListItemFunc = func(ctx context.Context, item *blocklistmodel.Item) error {
		return nil
	}

	// when
	rep := cachedBlockListRep{
		c:   cacheMock,
		rep: repMock,
	}

	err := rep.UpsertBlockListItem(context.Background(), &blocklistmodel.Item{
		Username: "ortuman",
		Jid:      "foo@jackal.im",
	})

	// then
	require.NoError(t, err)
	require.Equal(t, blockListNS("ortuman"), cacheNS)
	require.Equal(t, blockListItems, cacheKey)
	require.Len(t, repMock.UpsertBlockListItemCalls(), 1)
}

func TestCachedBlockListRep_DeleteBlockListItem(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteBlockListItemFunc = func(ctx context.Context, item *blocklistmodel.Item) error {
		return nil
	}

	// when
	rep := cachedBlockListRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteBlockListItem(context.Background(), &blocklistmodel.Item{
		Username: "ortuman",
		Jid:      "foo@jackal.im",
	})

	// then
	require.NoError(t, err)
	require.Equal(t, blockListNS("ortuman"), cacheNS)
	require.Equal(t, blockListItems, cacheKey)
	require.Len(t, repMock.DeleteBlockListItemCalls(), 1)
}

func TestCachedBlockListRep_DeleteBlockListItems(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteBlockListItemsFunc = func(ctx context.Context, username string) error {
		return nil
	}

	// when
	rep := cachedBlockListRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteBlockListItems(context.Background(), "ortuman")

	// then
	require.NoError(t, err)
	require.Equal(t, blockListNS("ortuman"), cacheNS)
	require.Len(t, repMock.DeleteBlockListItemsCalls(), 1)
}

func TestCachedBlockListRep_FetchBlockList(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchBlockListItemsFunc = func(ctx context.Context, username string) ([]*blocklistmodel.Item, error) {
		return []*blocklistmodel.Item{
			{Username: "ortuman", Jid: "foo@jackal.im"},
		}, nil
	}

	// when
	rep := cachedBlockListRep{
		c:   cacheMock,
		rep: repMock,
	}
	bl, err := rep.FetchBlockListItems(context.Background(), "ortuman")

	// then
	require.NotNil(t, bl)
	require.NoError(t, err)

	require.Len(t, bl, 1)
	require.Equal(t, "ortuman", bl[0].Username)
	require.Equal(t, "foo@jackal.im", bl[0].Jid)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchBlockListItemsCalls(), 1)
}
