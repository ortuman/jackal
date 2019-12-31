/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0163

import (
	"context"
	"reflect"
	"testing"

	pubsubmodel "github.com/ortuman/jackal/model/pubsub"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestDiscoInfoProvider_Identities(t *testing.T) {
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "yard", true)

	dp := &discoInfoProvider{}

	ids := dp.Identities(context.Background(), j1, j2, "")
	require.Len(t, ids, 2)

	require.Equal(t, "collection", ids[0].Type)
	require.Equal(t, "pubsub", ids[0].Category)
	require.Equal(t, "pep", ids[1].Type)
	require.Equal(t, "pubsub", ids[1].Category)

	ids = dp.Identities(context.Background(), j1, j2, "node")
	require.Len(t, ids, 2)

	require.Equal(t, "leaf", ids[0].Type)
	require.Equal(t, "pubsub", ids[0].Category)
	require.Equal(t, "pep", ids[1].Type)
	require.Equal(t, "pubsub", ids[1].Category)
}

func TestDiscoInfoProvider_Items(t *testing.T) {
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "yard", true)

	s := memory.New()

	storage.Set(s)
	defer storage.Unset()

	_ = s.UpsertNode(context.Background(), &pubsubmodel.Node{
		Host:    "ortuman@jackal.im",
		Name:    "princely_musings",
		Options: defaultNodeOptions,
	})
	dp := &discoInfoProvider{}

	items, err := dp.Items(context.Background(), j1, j2, "")
	require.Nil(t, items)
	require.NotNil(t, err)
	require.Equal(t, xmpp.ErrSubscriptionRequired, err)

	_, _ = s.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "noelia",
		JID:          "ortuman@jackal.im",
		Subscription: rostermodel.SubscriptionTo,
	})

	items, err = dp.Items(context.Background(), j1, j2, "")
	require.Nil(t, err)
	require.Len(t, items, 1)

	require.Equal(t, "ortuman@jackal.im", items[0].Jid)
	require.Equal(t, "princely_musings", items[0].Node)
}

func TestDiscoInfoProvider_Features(t *testing.T) {
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "yard", true)

	dp := &discoInfoProvider{}

	features, _ := dp.Features(context.Background(), j1, j2, "")
	require.True(t, reflect.DeepEqual(features, pepFeatures))

	features, _ = dp.Features(context.Background(), j1, j2, "node")
	require.True(t, reflect.DeepEqual(features, pepFeatures))
}

func TestDiscoInfoProvider_Form(t *testing.T) {
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "yard", true)

	dp := &discoInfoProvider{}

	features, _ := dp.Features(context.Background(), j1, j2, "")
	require.True(t, reflect.DeepEqual(features, pepFeatures))

	form, _ := dp.Form(context.Background(), j1, j2, "")
	require.Nil(t, form)

	form, _ = dp.Form(context.Background(), j1, j2, "node")
	require.Nil(t, form)
}
