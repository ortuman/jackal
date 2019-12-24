/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"context"
	"sort"
	"testing"

	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestServerProvider_Features(t *testing.T) {
	var sp serverProvider
	sp.registerServerFeature("sf0")
	sp.registerServerFeature("sf1")
	sp.registerServerFeature("sf1")
	sp.registerAccountFeature("af0")
	sp.registerAccountFeature("af1")
	sp.registerAccountFeature("af1")
	require.Equal(t, sp.serverFeatures, []Feature{"sf0", "sf1"})
	require.Equal(t, sp.accountFeatures, []Feature{"af0", "af1"})

	sp.unregisterServerFeature("sf1")
	sp.unregisterAccountFeature("af0")
	require.Equal(t, sp.serverFeatures, []Feature{"sf0"})
	require.Equal(t, sp.accountFeatures, []Feature{"af1"})

	srvJID, _ := jid.New("", "jackal.im", "", true)
	accJID, _ := jid.New("ortuman", "jackal.im", "garden", true)
	accJID2, _ := jid.New("noelia", "jackal.im", "balcony", true)

	features, sErr := sp.Features(srvJID, accJID, "node")
	require.Nil(t, features)
	require.Nil(t, sErr)

	features, sErr = sp.Features(srvJID, accJID, "")
	require.Equal(t, features, []Feature{"sf0"})
	require.Nil(t, sErr)

	features, sErr = sp.Features(accJID.ToBareJID(), accJID, "")
	require.Equal(t, features, []Feature{"af1"})
	require.Nil(t, sErr)

	features, sErr = sp.Features(accJID2.ToBareJID(), accJID, "")
	require.Nil(t, features)
	require.Equal(t, sErr, xmpp.ErrSubscriptionRequired)
}

func TestServerProvider_Identities(t *testing.T) {
	var sp serverProvider

	srvJID, _ := jid.New("", "jackal.im", "", true)
	accJID, _ := jid.New("ortuman", "jackal.im", "garden", true)
	require.Nil(t, sp.Identities(srvJID, accJID, "node"))

	require.Equal(t, sp.Identities(srvJID, accJID, ""), []Identity{
		{Type: "im", Category: "server", Name: "jackal"},
	})
	require.Equal(t, sp.Identities(accJID.ToBareJID(), accJID, ""), []Identity{
		{Type: "registered", Category: "account"},
	})
}

func TestServerProvider_Items(t *testing.T) {
	r, _, shutdown := setupTest("jackal.im")
	defer shutdown()

	var sp serverProvider
	sp.router = r

	srvJID, _ := jid.New("", "jackal.im", "", true)
	accJID1, _ := jid.New("ortuman", "jackal.im", "garden", true)
	accJID2, _ := jid.New("noelia", "jackal.im", "balcony", true)
	accJID3, _ := jid.New("noelia", "jackal.im", "yard", true)

	stm1 := stream.NewMockC2S(uuid.New(), accJID1)
	stm2 := stream.NewMockC2S(uuid.New(), accJID2)
	stm3 := stream.NewMockC2S(uuid.New(), accJID3)

	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)
	r.Bind(context.Background(), stm3)

	items, sErr := sp.Items(srvJID, accJID1, "node")
	require.Nil(t, items)
	require.Nil(t, sErr)

	items, sErr = sp.Items(srvJID, accJID1, "")
	require.Equal(t, items, []Item{
		{Jid: accJID1.ToBareJID().String()},
	})
	require.Nil(t, sErr)

	items, sErr = sp.Items(accJID2.ToBareJID(), accJID1, "")
	require.Nil(t, items)
	require.Equal(t, sErr, xmpp.ErrSubscriptionRequired)

	_, _ = storage.UpsertRosterItem(&rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})
	items, sErr = sp.Items(accJID2.ToBareJID(), accJID1, "")
	sort.Slice(items, func(i, j int) bool { return items[i].Jid < items[j].Jid })

	require.Equal(t, items, []Item{
		{Jid: accJID2.String()}, {Jid: accJID3.String()},
	})
	require.Nil(t, sErr)
}
