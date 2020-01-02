/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"
	"reflect"
	"testing"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_PubSubNodes(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	node := pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: pubsubmodel.Options{NotifySub: true},
	}
	err := h.db.UpsertNode(context.Background(), &node)
	require.Nil(t, err)

	sNode, err := h.db.FetchNode(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(sNode, &node))

	node2 := pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings_2",
		Options: pubsubmodel.Options{NotifySub: true},
	}
	err = h.db.UpsertNode(context.Background(), &node2)
	require.Nil(t, err)

	node3 := pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings_3",
		Options: pubsubmodel.Options{NotifySub: true},
	}
	err = h.db.UpsertNode(context.Background(), &node3)
	require.Nil(t, err)

	node4 := pubsubmodel.Node{
		Host:    "noelia@jackal.im",
		Name:    "princely_musings_1",
		Options: pubsubmodel.Options{NotifySub: true},
	}
	err = h.db.UpsertNode(context.Background(), &node4)
	require.Nil(t, err)

	nodes, err := h.db.FetchNodes(context.Background(), "ortuman@jackal.im")
	require.Nil(t, err)
	require.Len(t, nodes, 3)
	require.Equal(t, "princely_musings", nodes[0].Name)
	require.Equal(t, "princely_musings_2", nodes[1].Name)
	require.Equal(t, "princely_musings_3", nodes[2].Name)

	err = h.db.DeleteNode(context.Background(), "ortuman@jackal.im", "princely_musings_2")
	require.Nil(t, err)

	nodes, err = h.db.FetchNodes(context.Background(), "ortuman@jackal.im")
	require.Nil(t, err)
	require.Len(t, nodes, 2)
	require.Equal(t, "princely_musings", nodes[0].Name)
	require.Equal(t, "princely_musings_3", nodes[1].Name)

	// fetch hosts
	hosts, err := h.db.FetchHosts(context.Background())
	require.Nil(t, err)
	require.Len(t, hosts, 2)
}

func TestBadgerDB_PubSubItems(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	require.Nil(t, h.db.UpsertNodeItem(context.Background(), &pubsubmodel.Item{
		ID: "1234",
	}, "ortuman@jackal.im", "princely_musings", 2))
	require.Nil(t, h.db.UpsertNodeItem(context.Background(), &pubsubmodel.Item{
		ID: "5678",
	}, "ortuman@jackal.im", "princely_musings", 2))
	require.Nil(t, h.db.UpsertNodeItem(context.Background(), &pubsubmodel.Item{
		ID: "91011",
	}, "ortuman@jackal.im", "princely_musings", 2))

	items, err := h.db.FetchNodeItems(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, items, 2)
	require.Equal(t, "5678", items[0].ID)
	require.Equal(t, "91011", items[1].ID)

	items, err = h.db.FetchNodeItemsWithIDs(context.Background(), "ortuman@jackal.im", "princely_musings", []string{"5678"})
	require.Nil(t, err)

	require.Len(t, items, 1)
	require.Equal(t, "5678", items[0].ID)
}

func TestBadgerDB_PubSubAffiliations(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	require.Nil(t, h.db.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: "owner",
	}, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, h.db.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "noelia@jackal.im",
		Affiliation: "publisher",
	}, "ortuman@jackal.im", "princely_musings"))

	affiliations, err := h.db.FetchNodeAffiliations(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, affiliations, 2)
	require.Equal(t, "ortuman@jackal.im", affiliations[0].JID)
	require.Equal(t, "noelia@jackal.im", affiliations[1].JID)

	// delete affiliation
	err = h.db.DeleteNodeAffiliation(context.Background(), "noelia@jackal.im", "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	affiliations, err = h.db.FetchNodeAffiliations(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, affiliations, 1)
	require.Equal(t, "ortuman@jackal.im", affiliations[0].JID)
}

func TestBadgerDB_PubSubSubscriptions(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	node := &pubsubmodel.Node{
		Host: "ortuman@jackal.im",
		Name: "princely_musings",
	}
	_ = h.db.UpsertNode(context.Background(), node)

	node2 := &pubsubmodel.Node{
		Host: "noelia@jackal.im",
		Name: "princely_musings",
	}
	_ = h.db.UpsertNode(context.Background(), node2)

	require.Nil(t, h.db.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		SubID:        "1234",
		JID:          "ortuman@jackal.im",
		Subscription: "subscribed",
	}, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, h.db.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		SubID:        "5678",
		JID:          "noelia@jackal.im",
		Subscription: "unsubscribed",
	}, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, h.db.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		SubID:        "1234",
		JID:          "ortuman@jackal.im",
		Subscription: "subscribed",
	}, "noelia@jackal.im", "princely_musings"))

	subscriptions, err := h.db.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, subscriptions, 2)
	require.Equal(t, "ortuman@jackal.im", subscriptions[0].JID)
	require.Equal(t, "noelia@jackal.im", subscriptions[1].JID)

	// fetch user subscribed nodes
	nodes, err := h.db.FetchSubscribedNodes(context.Background(), "ortuman@jackal.im")
	require.Nil(t, err)
	require.Len(t, nodes, 2)

	// delete subscription
	err = h.db.DeleteNodeSubscription(context.Background(), "noelia@jackal.im", "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	subscriptions, err = h.db.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, subscriptions, 1)
	require.Equal(t, "ortuman@jackal.im", subscriptions[0].JID)
}
