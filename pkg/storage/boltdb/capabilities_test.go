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

	capsmodel "github.com/ortuman/jackal/pkg/model/caps"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestBoltDB_UpsertAndFetchCapabilities(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBCapsRep{tx: tx}

		err := rep.UpsertCapabilities(context.Background(), &capsmodel.Capabilities{
			Node: "n1",
			Ver:  "v1",
		})
		require.NoError(t, err)

		caps, err := rep.FetchCapabilities(context.Background(), "n1", "v1")
		require.NoError(t, err)

		require.Equal(t, "n1", caps.Node)
		require.Equal(t, "v1", caps.Ver)
		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_CapabilitiesExists(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBCapsRep{tx: tx}

		err := rep.UpsertCapabilities(context.Background(), &capsmodel.Capabilities{
			Node: "n1",
			Ver:  "v1",
		})
		require.NoError(t, err)

		ok, err := rep.CapabilitiesExist(context.Background(), "n1", "v1")
		require.NoError(t, err)

		require.True(t, ok)
		return nil
	})
	require.NoError(t, err)
}
