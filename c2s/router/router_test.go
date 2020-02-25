/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2srouter

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/router"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestRouter_Binding(t *testing.T) {
	j1, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	j2, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)

	stm1 := stream.NewMockC2S("id-1", j1)
	stm2 := stream.NewMockC2S("id-1", j2)

	r, _, _ := setupTest()

	r.Bind(stm1)
	r.Bind(stm2)
	stm1.SetPresence(xmpp.NewPresence(j1.ToBareJID(), j1, xmpp.AvailableType))
	stm2.SetPresence(xmpp.NewPresence(j2.ToBareJID(), j2, xmpp.AvailableType))

	require.Len(t, r.Streams("ortuman"), 2)

	require.NotNil(t, r.Stream("ortuman", "yard"))
	require.NotNil(t, r.Stream("ortuman", "balcony"))

	r.Unbind("ortuman", "yard")
	r.Unbind("ortuman", "balcony")

	require.Len(t, r.Streams("ortuman"), 0)

	r.(*c2sRouter).mu.RLock()
	require.Len(t, r.(*c2sRouter).tbl, 0)
	r.(*c2sRouter).mu.RUnlock()
}

func TestRouter_Routing(t *testing.T) {
	j1, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	j2, _ := jid.NewWithString("romeo@jackal.im/deadlyresource", true)
	stm1 := stream.NewMockC2S("id-1", j1)
	stm2 := stream.NewMockC2S("id-2", j2)

	r, userRep, blockListRep := setupTest()

	err := r.Route(context.Background(), xmpp.NewPresence(j1, j1, xmpp.AvailableType), true)
	require.Equal(t, router.ErrNotExistingAccount, err)

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "ortuman"})
	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "romeo"})

	err = r.Route(context.Background(), xmpp.NewPresence(j1, j1, xmpp.AvailableType), true)
	require.Equal(t, router.ErrNotAuthenticated, err)

	r.Bind(stm1)
	stm1.SetPresence(xmpp.NewPresence(j1.ToBareJID(), j1, xmpp.AvailableType))

	err = r.Route(context.Background(), xmpp.NewPresence(j1, j1, xmpp.AvailableType), true)
	require.Nil(t, err)

	// block jid
	r.Bind(stm2)
	stm2.SetPresence(xmpp.NewPresence(j2.ToBareJID(), j2, xmpp.AvailableType))

	_ = blockListRep.InsertBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "jackal.im/deadlyresource",
	})

	err = r.Route(context.Background(), xmpp.NewPresence(j1.ToBareJID(), j2, xmpp.AvailableType), true)
	require.Equal(t, router.ErrBlockedJID, err)
}

func TestRouter_PresencesMatching(t *testing.T) {
	j1, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	j2, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)
	j3, _ := jid.NewWithString("noelia@jackal.im/balcony", true)
	j4, _ := jid.NewWithString("noelia@jackal.im/chamber", true)
	j5, _ := jid.NewWithString("noelia@jackal.im/hall", true)

	stm1 := stream.NewMockC2S("id-1", j1)
	stm2 := stream.NewMockC2S("id-2", j2)
	stm3 := stream.NewMockC2S("id-3", j3)
	stm4 := stream.NewMockC2S("id-4", j4)
	stm5 := stream.NewMockC2S("id-5", j5)

	r, _, _ := setupTest()

	r.Bind(stm1)
	r.Bind(stm2)
	r.Bind(stm3)
	r.Bind(stm4)
	r.Bind(stm5)

	stm1.SetPresence(xmpp.NewPresence(j1.ToBareJID(), j1, xmpp.AvailableType))
	stm2.SetPresence(xmpp.NewPresence(j2.ToBareJID(), j2, xmpp.AvailableType))
	stm3.SetPresence(xmpp.NewPresence(j3.ToBareJID(), j3, xmpp.AvailableType))
	stm4.SetPresence(xmpp.NewPresence(j4.ToBareJID(), j4, xmpp.AvailableType))
	stm5.SetPresence(xmpp.NewPresence(j5.ToBareJID(), j5, xmpp.AvailableType))

	require.Len(t, r.PresencesMatching("ortuman", ""), 2)
	require.Len(t, r.PresencesMatching("noelia", ""), 3)
	require.Len(t, r.PresencesMatching("noelia", "chamber"), 1)
	require.Len(t, r.PresencesMatching("", "balcony"), 2)
	require.Len(t, r.PresencesMatching("", "hall"), 1)
	require.Len(t, r.PresencesMatching("", ""), 5)
}

func setupTest() (router.C2SRouter, repository.User, repository.BlockList) {
	userRep := memorystorage.NewUser()
	blockListRep := memorystorage.NewBlockList()
	return New(userRep, blockListRep), userRep, blockListRep
}
