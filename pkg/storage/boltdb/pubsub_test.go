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

	pubsubmodel "github.com/ortuman/jackal/pkg/model/pubsub"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestBoltDB_PubSubNode(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBPubSubRep{tx: tx}

		err := rep.UpsertNode(context.Background(), &pubsubmodel.Node{
			Host: "h0",
			Name: "princely_musings_0",
		})
		require.NoError(t, err)

		err = rep.UpsertNode(context.Background(), &pubsubmodel.Node{
			Host: "h0",
			Name: "princely_musings_1",
		})
		require.NoError(t, err)

		err = rep.UpsertNode(context.Background(), &pubsubmodel.Node{
			Host: "h0",
			Name: "princely_musings_2",
		})
		require.NoError(t, err)

		node, err := rep.FetchNode(context.Background(), "h0", "princely_musings_1")
		require.NoError(t, err)
		require.NotNil(t, node)

		nodes, err := rep.FetchNodes(context.Background(), "h0")
		require.NoError(t, err)

		require.Len(t, nodes, 3)

		require.NoError(t, rep.DeleteNode(context.Background(), "h0", "princely_musings_0"))

		nodes, err = rep.FetchNodes(context.Background(), "h0")
		require.NoError(t, err)
		require.Len(t, nodes, 2)

		require.NoError(t, rep.DeleteNodes(context.Background(), "h0"))

		nodes, err = rep.FetchNodes(context.Background(), "h0")
		require.NoError(t, err)
		require.Len(t, nodes, 0)

		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_PubSubAffiliation(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBPubSubRep{tx: tx}

		err := rep.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
			Jid:   "ortuman@jackal.im",
			State: pubsubmodel.AffiliationState_AFF_MEMBER,
		}, "h0", "princely_musings_0")
		require.NoError(t, err)

		err = rep.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
			Jid:   "noelia@jackal.im",
			State: pubsubmodel.AffiliationState_AFF_MEMBER,
		}, "h0", "princely_musings_0")
		require.NoError(t, err)

		err = rep.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
			Jid:   "romeo@jackal.im",
			State: pubsubmodel.AffiliationState_AFF_MEMBER,
		}, "h0", "princely_musings_0")
		require.NoError(t, err)

		aff, err := rep.FetchNodeAffiliation(context.Background(), "noelia@jackal.im", "h0", "princely_musings_0")
		require.NoError(t, err)
		require.NotNil(t, aff)

		affs, err := rep.FetchNodeAffiliations(context.Background(), "h0", "princely_musings_0")
		require.NoError(t, err)

		require.Len(t, affs, 3)

		require.NoError(t, rep.DeleteNodeAffiliation(context.Background(), "romeo@jackal.im", "h0", "princely_musings_0"))

		affs, err = rep.FetchNodeAffiliations(context.Background(), "h0", "princely_musings_0")
		require.NoError(t, err)
		require.Len(t, affs, 2)

		require.NoError(t, rep.DeleteNodeAffiliations(context.Background(), "h0", "princely_musings_0"))

		affs, err = rep.FetchNodeAffiliations(context.Background(), "h0", "princely_musings_0")
		require.NoError(t, err)
		require.Len(t, affs, 0)

		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_PubSubSubscription(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBPubSubRep{tx: tx}

		err := rep.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
			Jid:   "ortuman@jackal.im",
			State: pubsubmodel.SubscriptionState_SUB_PENDING,
		}, "h0", "princely_musings_0")
		require.NoError(t, err)

		err = rep.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
			Jid:   "noelia@jackal.im",
			State: pubsubmodel.SubscriptionState_SUB_PENDING,
		}, "h0", "princely_musings_0")
		require.NoError(t, err)

		err = rep.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
			Jid:   "romeo@jackal.im",
			State: pubsubmodel.SubscriptionState_SUB_PENDING,
		}, "h0", "princely_musings_0")
		require.NoError(t, err)

		sub, err := rep.FetchNodeSubscription(context.Background(), "noelia@jackal.im", "h0", "princely_musings_0")
		require.NoError(t, err)
		require.NotNil(t, sub)

		subs, err := rep.FetchNodeSubscriptions(context.Background(), "h0", "princely_musings_0")
		require.NoError(t, err)

		require.Len(t, subs, 3)

		require.NoError(t, rep.DeleteNodeSubscription(context.Background(), "romeo@jackal.im", "h0", "princely_musings_0"))

		subs, err = rep.FetchNodeSubscriptions(context.Background(), "h0", "princely_musings_0")
		require.NoError(t, err)
		require.Len(t, subs, 2)

		require.NoError(t, rep.DeleteNodeSubscriptions(context.Background(), "h0", "princely_musings_0"))

		subs, err = rep.FetchNodeSubscriptions(context.Background(), "h0", "princely_musings_0")
		require.NoError(t, err)
		require.Len(t, subs, 0)

		return nil
	})
	require.NoError(t, err)
}

func TestBoltDB_PubSubItem(t *testing.T) {
	t.Parallel()

	db := setupDB(t)
	t.Cleanup(func() { cleanUp(db) })

	err := db.Update(func(tx *bolt.Tx) error {
		rep := boltDBPubSubRep{tx: tx}

		m0 := testMessageStanza()
		m1 := testMessageStanza()
		m2 := testMessageStanza()

		err := rep.InsertNodeItem(context.Background(), &pubsubmodel.Item{
			Id:      "1",
			Payload: m0.Proto(),
		}, "h0", "princely_musings_0")
		require.NoError(t, err)

		err = rep.InsertNodeItem(context.Background(), &pubsubmodel.Item{
			Id:      "2",
			Payload: m1.Proto(),
		}, "h0", "princely_musings_0")
		require.NoError(t, err)

		err = rep.InsertNodeItem(context.Background(), &pubsubmodel.Item{
			Id:      "3",
			Payload: m2.Proto(),
		}, "h0", "princely_musings_0")
		require.NoError(t, err)

		items, err := rep.FetchNodeItems(context.Background(), "h0", "princely_musings_0")
		require.NoError(t, err)
		require.Len(t, items, 3)

		err = rep.DeleteOldestNodeItems(context.Background(), "h0", "princely_musings_0", 2)
		require.NoError(t, err)

		items, err = rep.FetchNodeItems(context.Background(), "h0", "princely_musings_0")
		require.NoError(t, err)
		require.Len(t, items, 2)

		err = rep.DeleteNodeItems(context.Background(), "h0", "princely_musings_0")
		require.NoError(t, err)

		items, err = rep.FetchNodeItems(context.Background(), "h0", "princely_musings_0")
		require.NoError(t, err)
		require.Len(t, items, 0)

		return nil
	})
	require.NoError(t, err)
}
