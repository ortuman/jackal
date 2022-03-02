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

	"github.com/jackal-xmpp/stravaganza"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestBoltDB_UpsertAndFetchVCard(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBVCardRep{tx: tx}

		vc0 := stravaganza.NewBuilder("vc").Build()

		err := rep.UpsertVCard(context.Background(), vc0, "ortuman")
		require.NoError(t, err)

		vc, err := rep.FetchVCard(context.Background(), "ortuman")
		require.NoError(t, err)

		require.Equal(t, "vc", vc.Name())
		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_DeleteVCard(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBVCardRep{tx: tx}

		vc := stravaganza.NewBuilder("vc").Build()

		err := rep.UpsertVCard(context.Background(), vc, "ortuman")
		require.NoError(t, err)

		err = rep.DeleteVCard(context.Background(), "ortuman")
		require.NoError(t, err)

		vc, err = rep.FetchVCard(context.Background(), "ortuman")
		require.NoError(t, err)

		require.Nil(t, vc)
		return nil
	})
	require.NoError(t, err)
}
