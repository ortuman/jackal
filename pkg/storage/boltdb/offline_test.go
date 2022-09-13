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

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestBoltDB_InsertAndFetchOfflineMessages(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBOfflineRep{tx: tx}

		m0 := testMessageStanza()
		m1 := testMessageStanza()

		err := rep.InsertOfflineMessage(context.Background(), m0, "ortuman")
		require.NoError(t, err)

		err = rep.InsertOfflineMessage(context.Background(), m1, "ortuman")
		require.NoError(t, err)

		messages, err := rep.FetchOfflineMessages(context.Background(), "ortuman")
		require.NoError(t, err)

		require.Len(t, messages, 2)

		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_CountOfflineMessages(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBOfflineRep{tx: tx}

		m0 := testMessageStanza()
		m1 := testMessageStanza()

		err := rep.InsertOfflineMessage(context.Background(), m0, "ortuman")
		require.NoError(t, err)

		err = rep.InsertOfflineMessage(context.Background(), m1, "ortuman")
		require.NoError(t, err)

		cnt, err := rep.CountOfflineMessages(context.Background(), "ortuman")
		require.NoError(t, err)

		require.Equal(t, 2, cnt)
		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_DeleteOfflineMessages(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBOfflineRep{tx: tx}

		m0 := testMessageStanza()
		m1 := testMessageStanza()

		err := rep.InsertOfflineMessage(context.Background(), m0, "ortuman")
		require.NoError(t, err)

		err = rep.InsertOfflineMessage(context.Background(), m1, "ortuman")
		require.NoError(t, err)

		cnt, err := rep.CountOfflineMessages(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Equal(t, 2, cnt)

		err = rep.DeleteOfflineMessages(context.Background(), "ortuman")
		require.NoError(t, err)

		cnt, err = rep.CountOfflineMessages(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Equal(t, 0, cnt)
		return nil
	})
	require.NoError(t, err)
}
