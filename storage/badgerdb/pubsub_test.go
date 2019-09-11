/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
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
	err := h.db.UpsertPubSubNode(&node)
	require.Nil(t, err)

	sNode, err := h.db.FetchPubSubNode("ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(sNode, &node))
}

func TestBadgerDB_PubSubItems(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	require.Nil(t, h.db.UpsertPubSubNodeItem(&pubsubmodel.Item{
		ID: "1234",
	}, "ortuman@jackal.im", "princely_musings", 2))
	require.Nil(t, h.db.UpsertPubSubNodeItem(&pubsubmodel.Item{
		ID: "5678",
	}, "ortuman@jackal.im", "princely_musings", 2))
	require.Nil(t, h.db.UpsertPubSubNodeItem(&pubsubmodel.Item{
		ID: "91011",
	}, "ortuman@jackal.im", "princely_musings", 2))

	items, err := h.db.FetchPubSubNodeItems("ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, items, 2)
	require.Equal(t, "5678", items[0].ID)
	require.Equal(t, "91011", items[1].ID)
}

func TestBadgerDB_PubSubAffiliations(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	require.Nil(t, h.db.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: "owner",
	}, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, h.db.UpsertPubSubNodeAffiliation(&pubsubmodel.Affiliation{
		JID:         "noelia@jackal.im",
		Affiliation: "publisher",
	}, "ortuman@jackal.im", "princely_musings"))

	affiliations, err := h.db.FetchPubSubNodeAffiliations("ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, affiliations, 2)
	require.Equal(t, "ortuman@jackal.im", affiliations[0].JID)
	require.Equal(t, "noelia@jackal.im", affiliations[1].JID)
}

func TestBadgerDB_PubSubSubscriptions(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	require.Nil(t, h.db.UpsertPubSubNodeSubscription(&pubsubmodel.Subscription{
		SubID:        "1234",
		JID:          "ortuman@jackal.im",
		Subscription: "subscribed",
	}, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, h.db.UpsertPubSubNodeSubscription(&pubsubmodel.Subscription{
		SubID:        "5678",
		JID:          "noelia@jackal.im",
		Subscription: "unsubscribed",
	}, "ortuman@jackal.im", "princely_musings"))

	subscriptions, err := h.db.FetchPubSubNodeSubscriptions("ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	require.Len(t, subscriptions, 2)
	require.Equal(t, "ortuman@jackal.im", subscriptions[0].JID)
	require.Equal(t, "noelia@jackal.im", subscriptions[1].JID)
}
