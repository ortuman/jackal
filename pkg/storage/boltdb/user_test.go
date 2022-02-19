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

	usermodel "github.com/ortuman/jackal/pkg/model/user"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestBoltDB_UpsertAndFetchUser(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBUserRep{tx: tx}

		err := rep.UpsertUser(context.Background(), &usermodel.User{
			Username: "ortuman",
		})
		require.NoError(t, err)

		usr, err := rep.FetchUser(context.Background(), "ortuman")
		require.NoError(t, err)

		require.Equal(t, "ortuman", usr.Username)
		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_UserExists(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBUserRep{tx: tx}

		err := rep.UpsertUser(context.Background(), &usermodel.User{
			Username: "ortuman",
		})
		require.NoError(t, err)

		ok, err := rep.UserExists(context.Background(), "ortuman")
		require.NoError(t, err)

		require.True(t, ok)
		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_DeleteUser(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBUserRep{tx: tx}

		err := rep.UpsertUser(context.Background(), &usermodel.User{
			Username: "ortuman",
		})
		require.NoError(t, err)

		err = rep.DeleteUser(context.Background(), "ortuman")
		require.NoError(t, err)

		ok, err := rep.UserExists(context.Background(), "ortuman")
		require.NoError(t, err)

		require.False(t, ok)
		return nil
	})
	require.NoError(t, err)
}
