/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"reflect"
	"testing"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestStorage_PubSubNode(t *testing.T) {
	s := New()
	node := &pubsubmodel.Node{
		Host: "ortuman@jackal.im",
		Name: "princely_musings",
	}
	require.Nil(t, s.UpsertPubSubNode(node))

	n, err := s.FetchPubSubNode("ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, n)

	require.True(t, reflect.DeepEqual(n, node))
}

func TestStorage_PubSubNodeItem(t *testing.T) {
	s := New()
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
	require.Nil(t, s.UpsertPubSubNodeItem(item1, "ortuman@jackal.im", "princely_musings", 1))
	require.Nil(t, s.UpsertPubSubNodeItem(item2, "ortuman@jackal.im", "princely_musings", 1))

	items, err := s.FetchPubSubNodeItems("ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, items)

	require.Len(t, items, 1)
	require.True(t, reflect.DeepEqual(&items[0], item2))

	// update item
	item2.Publisher = "ortuman@jackal.im"
	require.Nil(t, s.UpsertPubSubNodeItem(item2, "ortuman@jackal.im", "princely_musings", 1))

	items, err = s.FetchPubSubNodeItems("ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, items)

	require.Len(t, items, 1)
	require.True(t, reflect.DeepEqual(&items[0], item2))
}

func TestStorage_PubSubNodeAffiliation(t *testing.T) {
	s := New()
	aff1 := &pubsubmodel.Affiliation{
		JID:         "ortuman@jackal.im",
		Affiliation: "publisher",
	}
	aff2 := &pubsubmodel.Affiliation{
		JID:         "noelia@jackal.im",
		Affiliation: "publisher",
	}
	require.Nil(t, s.UpsertPubSubNodeAffiliation(aff1, "ortuman@jackal.im", "princely_musings"))
	require.Nil(t, s.UpsertPubSubNodeAffiliation(aff2, "ortuman@jackal.im", "princely_musings"))

	affiliations, err := s.FetchPubSubNodeAffiliations("ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, affiliations)

	require.Len(t, affiliations, 2)

	// update affiliation
	aff2.Affiliation = "owner"
	require.Nil(t, s.UpsertPubSubNodeAffiliation(aff2, "ortuman@jackal.im", "princely_musings"))

	affiliations, err = s.FetchPubSubNodeAffiliations("ortuman@jackal.im", "princely_musings")
	require.Nil(t, err)
	require.NotNil(t, affiliations)

	require.Len(t, affiliations, 2)

	for _, aff := range affiliations {
		if aff.JID == "noelia@jackal.im" {
			require.Equal(t, "owner", aff.Affiliation)
			return
		}
	}
	require.Fail(t, "affiliation for 'noelia@jackal.im' not found")
}
