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
	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	stm := stream.NewMockC2S("id-1", j)

	r, userRep, _ := setupTest()

	err := r.Route(context.Background(), xmpp.NewPresence(j, j, xmpp.AvailableType), true)
	require.Equal(t, router.ErrNotExistingAccount, err)

	_ = userRep.UpsertUser(context.Background(), &model.User{Username: "ortuman"})

	err = r.Route(context.Background(), xmpp.NewPresence(j, j, xmpp.AvailableType), true)
	require.Equal(t, router.ErrNotAuthenticated, err)

	r.Bind(stm)
	stm.SetPresence(xmpp.NewPresence(j.ToBareJID(), j, xmpp.AvailableType))

	err = r.Route(context.Background(), xmpp.NewPresence(j, j, xmpp.AvailableType), true)
	require.Nil(t, err)
}

func setupTest() (router.C2SRouter, repository.User, repository.BlockList) {
	userRep := memorystorage.NewUser()
	blockListRep := memorystorage.NewBlockList()
	return New(userRep, blockListRep), userRep, blockListRep
}
