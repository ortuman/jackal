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

	s, teardown := newPubSubMock()
	defer teardown()

	node := pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: pubsubmodel.Options{NotifySub: true},
	}
	err := s.UpsertNode(context.Background(), &node)
	require.Nil(t, err)

	sNode, err := s.FetchNode(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(sNode, &node))

	node2 := pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings_2",
		Options: pubsubmodel.Options{NotifySub: true},
	}
	err = s.UpsertNode(context.Background(), &node2)
	require.Nil(t, err)

	node3 := pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings_3",
		Options: pubsubmodel.Options{NotifySub: true},
	}
	err = s.UpsertNode(context.Background(), &node3)
	require.Nil(t, err)

	node4 := pubsubmodel.Node{
		Host:    "noelia@jackal.im",
		Name:    "princely_musings_1",
		Options: pubsubmodel.Options{NotifySub: true},
	}
	err = s.UpsertNode(context.Background(), &node4)
	require.Nil(t, err)

	nodes, err := s.FetchNodes(context.Background(), "ortuman@jackal.im")
	require.Nil(t, err)
	require.Len(t, nodes, 3)
	require.Equal(t, "princely_musings", nodes[0].Name)
	require.Equal(t, "princely_musings_2", nodes[1].Name)
	require.Equal(t, "princely_musings_3", nodes[2].Name)

	err = s.DeleteNode(context.Background(), "ortuman@jackal.im", "princely_musings_2")
	require.Nil(t, err)

	nodes, err = s.FetchNodes(context.Background(), "ortuman@jackal.im")
	require.Nil(t, err)
	require.Len(t, nodes, 2)
	require.Equal(t, "princely_musings", nodes[0].Name)
	require.Equal(t, "princely_musings_3", nodes[1].Name)

	// fetch hosts
	hosts, err := s.FetchHosts(context.Background())
	require.Nil(t, err)
	require.Len(t, hosts, 2)
}

func TestBadgerDB_PubSubItems(t *testing.T) {
	t.Parallel()

	s, teardown := newPubSubMock()
	defer teardown()

	require.Nil(t, s.UpsertNodeItem(context.Background(), &pubsubmodel.Item{
		ID: "1234",
	}, "ortuman@jackal.im", "princely_musings", 2))
	require.Nil(t, s.UpsertNodeItem(context.Background(), &pubsubmodel.Item{
		ID: "5678",
	}, "ortuman@jackal.im", "princely_musings", 2))
	require.Nil(t, s.UpsertNodeItem(context.Background(), &pubsubmodel.Item{
		ID: "91011",
	}, "ortuman@jackal.im", "princely_musings", 2))

	items, err := s.FetchNodeItems(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, items, 2)
	require.Equal(t, "5678", items[0].ID)
	require.Equal(t, "91011", items[1].ID)

	items, err = s.FetchNodeItemsWithIDs(context.Background(), "ortuman@jackal.im", "princely_musings", []string{"5678"})
	require.Nil(t, err)

	require.Len(t, items, 1)
	require.Equal(t, "5678", items[0].ID)
}

func TestBadgerDB_PubSubAffiliations(t *testing.T) {
	t.Parallel()

	s, teardown := newPubSubMock()
	defer teardown()

	require.Nil(t, s.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: "owner",
	}, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, s.UpsertNodeAffiliation(context.Background(), &pubsubmodel.Affiliation{
		JID:         "noelia@jackal.im",
		Affiliation: "publisher",
	}, "ortuman@jackal.im", "princely_musings"))

	affiliations, err := s.FetchNodeAffiliations(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, affiliations, 2)
	require.Equal(t, "ortuman@jackal.im", affiliations[0].JID)
	require.Equal(t, "noelia@jackal.im", affiliations[1].JID)

	// delete affiliation
	err = s.DeleteNodeAffiliation(context.Background(), "noelia@jackal.im", "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	affiliations, err = s.FetchNodeAffiliations(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, affiliations, 1)
	require.Equal(t, "ortuman@jackal.im", affiliations[0].JID)
}

func TestBadgerDB_PubSubSubscriptions(t *testing.T) {
	t.Parallel()

	s, teardown := newPubSubMock()
	defer teardown()

	node := &pubsubmodel.Node{
		Host: "ortuman@jackal.im",
		Name: "princely_musings",
	}
	_ = s.UpsertNode(context.Background(), node)

	node2 := &pubsubmodel.Node{
		Host: "noelia@jackal.im",
		Name: "princely_musings",
	}
	_ = s.UpsertNode(context.Background(), node2)

	require.Nil(t, s.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		SubID:        "1234",
		JID:          "ortuman@jackal.im",
		Subscription: "subscribed",
	}, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, s.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		SubID:        "5678",
		JID:          "noelia@jackal.im",
		Subscription: "unsubscribed",
	}, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, s.UpsertNodeSubscription(context.Background(), &pubsubmodel.Subscription{
		SubID:        "1234",
		JID:          "ortuman@jackal.im",
		Subscription: "subscribed",
	}, "noelia@jackal.im", "princely_musings"))

	subscriptions, err := s.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, subscriptions, 2)
	require.Equal(t, "ortuman@jackal.im", subscriptions[0].JID)
	require.Equal(t, "noelia@jackal.im", subscriptions[1].JID)

	// fetch user subscribed nodes
	nodes, err := s.FetchSubscribedNodes(context.Background(), "ortuman@jackal.im")
	require.Nil(t, err)
	require.Len(t, nodes, 2)

	// delete subscription
	err = s.DeleteNodeSubscription(context.Background(), "noelia@jackal.im", "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	subscriptions, err = s.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, subscriptions, 1)
	require.Equal(t, "ortuman@jackal.im", subscriptions[0].JID)
}

func newPubSubMock() (*badgerDBPubSub, func()) {
	t := newT()
	return &badgerDBPubSub{badgerDBStorage: newStorage(t.db)}, func() {
		t.teardown()
	}
}
