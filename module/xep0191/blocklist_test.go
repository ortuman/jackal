// Copyright 2021 The jackal Authors
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

package xep0191

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	blocklistmodel "github.com/ortuman/jackal/model/blocklist"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/router/stream"
	"github.com/stretchr/testify/require"
)

func TestBlockList_GetBlockList(t *testing.T) {
	// given
	routerMock := &routerMock{}
	rep := &repositoryMock{}
	stmMock := &c2sStreamMock{}

	var setK, setVal string
	stmMock.SetValueFunc = func(ctx context.Context, k string, val string) error {
		setK = k
		setVal = val
		return nil
	}
	c2sRouterMock := &c2sRouterMock{}
	c2sRouterMock.LocalStreamFunc = func(username string, resource string) stream.C2S {
		return stmMock
	}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	routerMock.C2SFunc = func() router.C2SRouter {
		return c2sRouterMock
	}

	rep.FetchBlockListItemsFunc = func(ctx context.Context, username string) ([]blocklistmodel.Item, error) {
		return []blocklistmodel.Item{
			{Username: "ortuman", JID: "noelia@jackal.im"},
			{Username: "ortuman", JID: "jabber.org"},
		}, nil
	}
	// when
	bl := &BlockList{
		router: routerMock,
		rep:    rep,
		sn:     sonar.New(),
	}

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, uuid.New().String()).
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/chamber").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("blocklist").
				WithAttribute(stravaganza.Namespace, blockListNamespace).
				Build(),
		).
		BuildIQ(false)

	// then
	_ = bl.ProcessIQ(context.Background(), iq)

	require.Len(t, respStanzas, 1)
	require.Equal(t, stravaganza.ResultType, respStanzas[0].Attribute(stravaganza.Type))

	blResp := respStanzas[0].ChildNamespace("blocklist", blockListNamespace)
	require.NotNil(t, bl)

	items := blResp.Children("item")
	require.Len(t, items, 2)

	require.Equal(t, "noelia@jackal.im", items[0].Attribute("jid"))
	require.Equal(t, "jabber.org", items[1].Attribute("jid"))

	require.Equal(t, setK, blockListRequestedCtxKey)
	require.Equal(t, setVal, "true")
}

func TestBlockList_BlockItem(t *testing.T) {
	// given
	// when
	// then
}

func TestBlockList_UnblockItem(t *testing.T) {
	// given
	// when
	// then
}

func TestBlockList_Forbidden(t *testing.T) {
	// given
	// when
	// then
}

func TestBlockList_UserDeleted(t *testing.T) {
	// given
	// when
	// then
}

func TestBlockList_InterceptIncomingStanza(t *testing.T) {
	// given
	// when
	// then
}

func TestBlockList_InterceptOutgoingStanza(t *testing.T) {
	// given
	// when
	// then
}

func TestBlockList_PresenceTargets(t *testing.T) {
	// given
	rep := &repositoryMock{}
	rep.FetchRosterItemsFunc = func(ctx context.Context, username string) ([]rostermodel.Item, error) {
		return []rostermodel.Item{
			{Username: "ortuman", JID: "juliet@jabber.org", Subscription: rostermodel.Both},
			{Username: "ortuman", JID: "hamlet@jabber.org", Subscription: rostermodel.To},
			{Username: "ortuman", JID: "hamlet@404.city", Subscription: rostermodel.Both},
			{Username: "ortuman", JID: "macbeth@404.city", Subscription: rostermodel.Both},
			{Username: "ortuman", JID: "witch@404.city", Subscription: rostermodel.To},
			{Username: "ortuman", JID: "witch@jackal.im", Subscription: rostermodel.Both},
			{Username: "ortuman", JID: "witch@jabber.net", Subscription: rostermodel.Both},
			{Username: "ortuman", JID: "witch@jabber.org", Subscription: rostermodel.To},
		}, nil
	}
	// when
	bl := &BlockList{
		rep: rep,
	}
	jd0, _ := jid.NewWithString("404.city/yard", true)
	jd1, _ := jid.NewWithString("jabber.org", true)
	jd2, _ := jid.NewWithString("witch@jackal.im", true)
	jd3, _ := jid.NewWithString("witch@jabber.net/chamber", true)

	pss, _ := bl.getPresenceTargets(context.Background(), []jid.JID{*jd0, *jd1, *jd2, *jd3}, "ortuman")

	// then
	require.Len(t, pss, 5)

	require.Equal(t, pss[0].String(), "hamlet@404.city/yard")
	require.Equal(t, pss[1].String(), "macbeth@404.city/yard")
	require.Equal(t, pss[2].String(), "juliet@jabber.org")
	require.Equal(t, pss[3].String(), "witch@jackal.im")
	require.Equal(t, pss[4].String(), "witch@jabber.net/chamber")
}
