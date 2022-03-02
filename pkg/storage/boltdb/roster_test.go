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

	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestBoltDB_TouchAndFetchRosterVersion(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBRosterRep{tx: tx}

		ver, err := rep.TouchRosterVersion(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Equal(t, 1, ver)

		ver, err = rep.FetchRosterVersion(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Equal(t, 1, ver)

		ver, err = rep.TouchRosterVersion(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Equal(t, 2, ver)
		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_RosterItems(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBRosterRep{tx: tx}

		err := rep.UpsertRosterItem(context.Background(), &rostermodel.Item{
			Username: "ortuman",
			Jid:      "foo@jackal.im",
			Groups:   []string{"g1"},
		})
		require.NoError(t, err)

		err = rep.UpsertRosterItem(context.Background(), &rostermodel.Item{
			Username: "ortuman",
			Jid:      "foo-2@jackal.im",
			Groups:   []string{"g2"},
		})
		require.NoError(t, err)

		itm, err := rep.FetchRosterItem(context.Background(), "ortuman", "foo@jackal.im")
		require.NoError(t, err)
		require.NotNil(t, itm)
		require.Equal(t, "foo@jackal.im", itm.Jid)

		items, err := rep.FetchRosterItems(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Len(t, items, 2)

		items, err = rep.FetchRosterItemsInGroups(context.Background(), "ortuman", []string{"g2"})
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, "foo-2@jackal.im", items[0].Jid)

		err = rep.DeleteRosterItem(context.Background(), "ortuman", "foo-2@jackal.im")
		require.NoError(t, err)

		items, err = rep.FetchRosterItems(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Len(t, items, 1)

		err = rep.DeleteRosterItems(context.Background(), "ortuman")
		require.NoError(t, err)

		items, err = rep.FetchRosterItems(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Len(t, items, 0)

		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_RosterNotifications(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBRosterRep{tx: tx}

		err := rep.UpsertRosterNotification(context.Background(), &rostermodel.Notification{
			Contact: "ortuman",
			Jid:     "foo-1@jackal.im",
		})
		require.NoError(t, err)

		err = rep.UpsertRosterNotification(context.Background(), &rostermodel.Notification{
			Contact: "ortuman",
			Jid:     "foo-2@jackal.im",
		})
		require.NoError(t, err)

		n, err := rep.FetchRosterNotification(context.Background(), "ortuman", "foo-1@jackal.im")
		require.NoError(t, err)
		require.NotNil(t, n)
		require.Equal(t, "foo-1@jackal.im", n.Jid)

		ns, err := rep.FetchRosterNotifications(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Len(t, ns, 2)

		err = rep.DeleteRosterNotification(context.Background(), "ortuman", "foo-2@jackal.im")
		require.NoError(t, err)

		ns, err = rep.FetchRosterNotifications(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Len(t, ns, 1)

		err = rep.DeleteRosterNotifications(context.Background(), "ortuman")
		require.NoError(t, err)

		ns, err = rep.FetchRosterNotifications(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Len(t, ns, 0)

		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_TouchAndFetchRosterGroups(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBRosterRep{tx: tx}

		err := rep.UpsertRosterItem(context.Background(), &rostermodel.Item{
			Username: "ortuman",
			Jid:      "foo@jackal.im",
			Groups:   []string{"g1"},
		})
		require.NoError(t, err)

		err = rep.UpsertRosterItem(context.Background(), &rostermodel.Item{
			Username: "ortuman",
			Jid:      "foo-2@jackal.im",
			Groups:   []string{"g2"},
		})
		require.NoError(t, err)

		groups, err := rep.FetchRosterGroups(context.Background(), "ortuman")
		require.NoError(t, err)
		require.Contains(t, groups, "g1")
		require.Contains(t, groups, "g2")

		return nil
	})
	require.NoError(t, err)
}
