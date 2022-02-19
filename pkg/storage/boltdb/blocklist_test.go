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

package boltdb

import (
	"context"
	"testing"

	blocklistmodel "github.com/ortuman/jackal/pkg/model/blocklist"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestBoltDB_UpsertAndFetchBlockListItems(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBBlockListRep{tx: tx}

		err := rep.UpsertBlockListItem(context.Background(), &blocklistmodel.Item{
			Username: "ortuman",
			Jid:      "foo-1@jackal.im",
		})
		require.NoError(t, err)

		err = rep.UpsertBlockListItem(context.Background(), &blocklistmodel.Item{
			Username: "ortuman",
			Jid:      "foo-2@jackal.im",
		})
		require.NoError(t, err)

		items, err := rep.FetchBlockListItems(context.Background(), "ortuman")
		require.NoError(t, err)

		require.Len(t, items, 2)

		require.Equal(t, "foo-1@jackal.im", items[0].Jid)
		require.Equal(t, "foo-2@jackal.im", items[1].Jid)
		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_DeleteBlockListItem(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBBlockListRep{tx: tx}

		err := rep.UpsertBlockListItem(context.Background(), &blocklistmodel.Item{
			Username: "ortuman",
			Jid:      "foo-1@jackal.im",
		})
		require.NoError(t, err)

		items, err := rep.FetchBlockListItems(context.Background(), "ortuman")
		require.NoError(t, err)

		require.Len(t, items, 1)

		err = rep.DeleteBlockListItem(context.Background(), &blocklistmodel.Item{
			Username: "ortuman",
			Jid:      "foo-1@jackal.im",
		})
		require.NoError(t, err)

		items, err = rep.FetchBlockListItems(context.Background(), "ortuman")
		require.NoError(t, err)

		require.Len(t, items, 0)
		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_DeleteBlockListItems(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBBlockListRep{tx: tx}

		err := rep.UpsertBlockListItem(context.Background(), &blocklistmodel.Item{
			Username: "ortuman",
			Jid:      "foo-1@jackal.im",
		})
		require.NoError(t, err)

		err = rep.UpsertBlockListItem(context.Background(), &blocklistmodel.Item{
			Username: "ortuman",
			Jid:      "foo-2@jackal.im",
		})
		require.NoError(t, err)

		items, err := rep.FetchBlockListItems(context.Background(), "ortuman")
		require.NoError(t, err)

		require.Len(t, items, 2)

		err = rep.DeleteBlockListItems(context.Background(), "ortuman")
		require.NoError(t, err)

		items, err = rep.FetchBlockListItems(context.Background(), "ortuman")
		require.NoError(t, err)

		require.Len(t, items, 0)
		return nil
	})
	require.NoError(t, err)
}
