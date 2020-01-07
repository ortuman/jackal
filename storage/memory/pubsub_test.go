/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"
	"reflect"
	"testing"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestStorage_PubSubNode(t *testing.T) {
	s := NewPubSub()
	node := &pubsubmodel.Node{
		Host: "ortuman@jackal.im",
		Name: "princely_musings",
	}
	require.Nil(t, s.UpsertNode(context.Background(), node))

	n, err := s.FetchNode(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, n)

	require.True(t, reflect.DeepEqual(n, node))

	node2 := &pubsubmodel.Node{
		Host: "ortuman@jackal.im",
		Name: "princely_musings_2",
	}
	node3 := &pubsubmodel.Node{
		Host: "ortuman@jackal.im",
		Name: "princely_musings_3",
	}
	node4 := &pubsubmodel.Node{
		Host: "noelia@jackal.im",
		Name: "princely_musings_1",
	}
	require.Nil(t, s.UpsertNode(context.Background(), node2))
	require.Nil(t, s.UpsertNode(context.Background(), node3))
	require.Nil(t, s.UpsertNode(context.Background(), node4))

	nodes, err := s.FetchNodes(context.Background(), "ortuman@jackal.im")
	require.Nil(t, err)
	require.NotNil(t, nodes)

	require.Len(t, nodes, 3)
	require.Equal(t, "princely_musings", nodes[0].Name)
	require.Equal(t, "princely_musings_2", nodes[1].Name)
	require.Equal(t, "princely_musings_3", nodes[2].Name)

	require.Nil(t, s.DeleteNode(context.Background(), "ortuman@jackal.im", "princely_musings_2"))

	nodes, err = s.FetchNodes(context.Background(), "ortuman@jackal.im")
	require.Nil(t, err)
	require.NotNil(t, nodes)

	require.Len(t, nodes, 2)
	require.Equal(t, "princely_musings", nodes[0].Name)
	require.Equal(t, "princely_musings_3", nodes[1].Name)

	// fetch hosts
	hosts, err := s.FetchHosts(context.Background())
	require.Nil(t, err)
	require.Len(t, hosts, 2)
}

func TestStorage_PubSubNodeItem(t *testing.T) {
	s := NewPubSub()
	item1 := &pubsubmodel.Item{
		ID:        "id1",
		Publisher: "ortuman@jackal.im",
		Payload:   xmpp.NewElementName("a"),
	}
	item2 := &pubsubmodel.Item{
		ID:        "id2",
		Publisher: "noelia@jackal.im",
		Payload:   xmpp.NewElementName("b"),
	}
	item3 := &pubsubmodel.Item{
		ID:        "id3",
		Publisher: "noelia@jackal.im",
		Payload:   xmpp.NewElementName("c"),
	}
	require.Nil(t, s.UpsertNodeItem(context.Background(), item1, "ortuman@jackal.im", "princely_musings", 1))
	require.Nil(t, s.UpsertNodeItem(context.Background(), item2, "ortuman@jackal.im", "princely_musings", 1))

	items, err := s.FetchNodeItems(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, items)

	require.Len(t, items, 1)
	require.True(t, reflect.DeepEqual(&items[0], item2))

	// update item
	require.Nil(t, s.UpsertNodeItem(context.Background(), item3, "ortuman@jackal.im", "princely_musings", 2))

	items, err = s.FetchNodeItems(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, items)

	require.Len(t, items, 2)
	require.True(t, reflect.DeepEqual(&items[0], item2))
	require.True(t, reflect.DeepEqual(&items[1], item3))

	items, err = s.FetchNodeItemsWithIDs(context.Background(), "ortuman@jackal.im", "princely_musings", []string{"id3"})
	require.Nil(t, err)
	require.NotNil(t, items)

	require.Len(t, items, 1)
	require.Equal(t, "id3", items[0].ID)
}

func TestStorage_PubSubNodeAffiliation(t *testing.T) {
	s := NewPubSub()
	aff1 := &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: "publisher",
	}
	aff2 := &pubsubmodel.Affiliation{
		JID:         "noelia@jackal.im",
		Affiliation: "publisher",
	}
	require.Nil(t, s.UpsertNodeAffiliation(context.Background(), aff1, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, s.UpsertNodeAffiliation(context.Background(), aff2, "ortuman@jackal.im", "princely_musings"))

	affiliations, err := s.FetchNodeAffiliations(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, affiliations)

	require.Len(t, affiliations, 2)

	// update affiliation
	aff2.Affiliation = "owner"
	require.Nil(t, s.UpsertNodeAffiliation(context.Background(), aff2, "ortuman@jackal.im", "princely_musings"))

	affiliations, err = s.FetchNodeAffiliations(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, affiliations)

	require.Len(t, affiliations, 2)

	var updated bool
	for _, aff := range affiliations {
		if aff.JID == "noelia@jackal.im" {
			require.Equal(t, "owner", aff.Affiliation)
			updated = true
			break
		}
	}
	if !updated {
		require.Fail(t, "affiliation for 'noelia@jackal.im' not found")
	}

	// delete affiliation
	err = s.DeleteNodeAffiliation(context.Background(), "noelia@jackal.im", "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	affiliations, err = s.FetchNodeAffiliations(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, affiliations)

	require.Len(t, affiliations, 1)
}

func TestStorage_PubSubNodeSubscription(t *testing.T) {
	s := NewPubSub()
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

	sub1 := &pubsubmodel.Subscription{
		SubID:        "1234",
		JID:          "ortuman@jackal.im",
		Subscription: "subscribed",
	}
	sub2 := &pubsubmodel.Subscription{
		SubID:        "5678",
		JID:          "noelia@jackal.im",
		Subscription: "unsubscribed",
	}
	sub3 := &pubsubmodel.Subscription{
		SubID:        "9012",
		JID:          "ortuman@jackal.im",
		Subscription: "subscribed",
	}
	require.Nil(t, s.UpsertNodeSubscription(context.Background(), sub1, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, s.UpsertNodeSubscription(context.Background(), sub2, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, s.UpsertNodeSubscription(context.Background(), sub3, "noelia@jackal.im", "princely_musings"))

	// fetch user subscribed nodes
	nodes, err := s.FetchSubscribedNodes(context.Background(), "ortuman@jackal.im")
	require.Nil(t, err)
	require.Len(t, nodes, 2)

	subscriptions, err := s.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, subscriptions)

	require.Len(t, subscriptions, 2)

	// update affiliation
	sub2.Subscription = "subscribed"
	require.Nil(t, s.UpsertNodeSubscription(context.Background(), sub2, "ortuman@jackal.im", "princely_musings"))

	subscriptions, err = s.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, subscriptions)

	require.Len(t, subscriptions, 2)

	var updated bool
	for _, sub := range subscriptions {
		if sub.JID == "noelia@jackal.im" {
			require.Equal(t, "subscribed", sub.Subscription)
			updated = true
			break
		}
	}
	if !updated {
		require.Fail(t, "subscription for 'noelia@jackal.im' not found")
	}

	// delete subscription
	err = s.DeleteNodeSubscription(context.Background(), "noelia@jackal.im", "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)

	subscriptions, err = s.FetchNodeSubscriptions(context.Background(), "ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, subscriptions)

	require.Len(t, subscriptions, 1)
}
