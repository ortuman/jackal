// Copyright 2022 The jackal Authors
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

	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/pkg/hook"
	blocklistmodel "github.com/ortuman/jackal/pkg/model/blocklist"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/ortuman/jackal/pkg/storage/repository"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
	"github.com/stretchr/testify/require"
)

func TestBlockList_GetBlockList(t *testing.T) {
	// given
	routerMock := &routerMock{}
	rep := &repositoryMock{}
	stmMock := &c2sStreamMock{}

	var setK string
	var setVal interface{}
	stmMock.SetInfoValueFunc = func(ctx context.Context, k string, val interface{}) error {
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

	rep.FetchBlockListItemsFunc = func(ctx context.Context, username string) ([]*blocklistmodel.Item, error) {
		return []*blocklistmodel.Item{
			{Username: "ortuman", Jid: "noelia@jackal.im"},
			{Username: "ortuman", Jid: "jabber.org"},
		}, nil
	}
	bl := &BlockList{
		router: routerMock,
		rep:    rep,
		hk:     hook.NewHooks(),
		logger: kitlog.NewNopLogger(),
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
		BuildIQ()

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
	require.Equal(t, setVal, true)
}

func TestBlockList_BlockItem(t *testing.T) {
	// given
	routerMock := &routerMock{}
	resMngMock := &resourceManagerMock{}
	rep := &repositoryMock{}
	txMock := &txMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	txMock.UpsertBlockListItemFunc = func(ctx context.Context, item *blocklistmodel.Item) error {
		return nil
	}
	rep.FetchRosterItemsFunc = func(ctx context.Context, username string) ([]*rostermodel.Item, error) {
		return []*rostermodel.Item{
			{Username: "ortuman", Jid: "juliet@jabber.org", Subscription: rostermodel.Both},
		}, nil
	}
	rep.FetchBlockListItemsFunc = func(ctx context.Context, username string) ([]*blocklistmodel.Item, error) {
		return nil, nil
	}
	rep.InTransactionFunc = func(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
		return f(ctx, txMock)
	}

	jd0, _ := jid.NewWithString("ortuman@jackal.im/chamber", true)
	jd1, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	resMngMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.ResourceDesc, error) {
		return []c2smodel.ResourceDesc{
			c2smodel.NewResourceDesc("inst-1", jd0, nil, c2smodel.NewInfoMapFromMap(map[string]string{blockListRequestedCtxKey: "true"})),
			c2smodel.NewResourceDesc("inst-1", jd1, nil, c2smodel.NewInfoMapFromMap(map[string]string{blockListRequestedCtxKey: "true"})),
		}, nil
	}
	bl := &BlockList{
		router: routerMock,
		rep:    rep,
		resMng: resMngMock,
		hk:     hook.NewHooks(),
		logger: kitlog.NewNopLogger(),
	}

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, uuid.New().String()).
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/chamber").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("block").
				WithAttribute(stravaganza.Namespace, blockListNamespace).
				WithChild(
					stravaganza.NewBuilder("item").
						WithAttribute("jid", "jabber.org").
						Build(),
				).
				Build(),
		).
		BuildIQ()

	// then
	_ = bl.ProcessIQ(context.Background(), iq)

	require.Len(t, respStanzas, 5)

	require.Equal(t, "presence", respStanzas[0].Name())
	require.Equal(t, "ortuman@jackal.im/chamber", respStanzas[0].Attribute(stravaganza.From))
	require.Equal(t, "juliet@jabber.org", respStanzas[0].Attribute(stravaganza.To))
	require.Equal(t, stravaganza.UnavailableType, respStanzas[0].Attribute(stravaganza.Type))

	require.Equal(t, "presence", respStanzas[1].Name())
	require.Equal(t, "ortuman@jackal.im/yard", respStanzas[1].Attribute(stravaganza.From))
	require.Equal(t, "juliet@jabber.org", respStanzas[1].Attribute(stravaganza.To))
	require.Equal(t, stravaganza.UnavailableType, respStanzas[1].Attribute(stravaganza.Type))

	require.Equal(t, "iq", respStanzas[2].Name())
	require.Equal(t, stravaganza.ResultType, respStanzas[2].Attribute(stravaganza.Type))

	require.Equal(t, "iq", respStanzas[3].Name())
	require.Equal(t, "ortuman@jackal.im", respStanzas[3].Attribute(stravaganza.From))
	require.Equal(t, "ortuman@jackal.im/chamber", respStanzas[3].Attribute(stravaganza.To))
	require.Equal(t, stravaganza.SetType, respStanzas[3].Attribute(stravaganza.Type))
	require.NotNil(t, respStanzas[3].ChildNamespace("block", blockListNamespace))

	require.Equal(t, "iq", respStanzas[4].Name())
	require.Equal(t, "ortuman@jackal.im", respStanzas[4].Attribute(stravaganza.From))
	require.Equal(t, "ortuman@jackal.im/yard", respStanzas[4].Attribute(stravaganza.To))
	require.Equal(t, stravaganza.SetType, respStanzas[4].Attribute(stravaganza.Type))
	require.NotNil(t, respStanzas[4].ChildNamespace("block", blockListNamespace))
}

func TestBlockList_UnblockItem(t *testing.T) {
	// given
	routerMock := &routerMock{}
	resMngMock := &resourceManagerMock{}
	rep := &repositoryMock{}
	txMock := &txMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	txMock.DeleteBlockListItemFunc = func(ctx context.Context, item *blocklistmodel.Item) error {
		return nil
	}
	rep.FetchRosterItemsFunc = func(ctx context.Context, username string) ([]*rostermodel.Item, error) {
		return []*rostermodel.Item{
			{Username: "ortuman", Jid: "juliet@jabber.org", Subscription: rostermodel.Both},
		}, nil
	}
	rep.FetchBlockListItemsFunc = func(ctx context.Context, username string) ([]*blocklistmodel.Item, error) {
		return []*blocklistmodel.Item{
			{Username: "ortuman", Jid: "jabber.org"},
		}, nil
	}
	rep.InTransactionFunc = func(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
		return f(ctx, txMock)
	}

	jd0, _ := jid.NewWithString("ortuman@jackal.im/chamber", true)
	jd1, _ := jid.NewWithString("ortuman@jackal.im/yard", true)

	resMngMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.ResourceDesc, error) {
		return []c2smodel.ResourceDesc{
			c2smodel.NewResourceDesc("i1", jd0, xmpputil.MakePresence(jd0.ToBareJID(), jd0, stravaganza.AvailableType, nil), c2smodel.NewInfoMapFromMap(map[string]string{blockListRequestedCtxKey: "true"})),
			c2smodel.NewResourceDesc("i1", jd1, xmpputil.MakePresence(jd1.ToBareJID(), jd1, stravaganza.AvailableType, nil), c2smodel.NewInfoMapFromMap(map[string]string{blockListRequestedCtxKey: "true"})),
		}, nil
	}
	bl := &BlockList{
		router: routerMock,
		rep:    rep,
		resMng: resMngMock,
		hk:     hook.NewHooks(),
		logger: kitlog.NewNopLogger(),
	}

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, uuid.New().String()).
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/chamber").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("unblock").
				WithAttribute(stravaganza.Namespace, blockListNamespace).
				WithChild(
					stravaganza.NewBuilder("item").
						WithAttribute("jid", "jabber.org").
						Build(),
				).
				Build(),
		).
		BuildIQ()

	// then
	_ = bl.ProcessIQ(context.Background(), iq)

	require.Len(t, respStanzas, 5)

	require.Equal(t, "presence", respStanzas[0].Name())
	require.Equal(t, "ortuman@jackal.im/chamber", respStanzas[0].Attribute(stravaganza.From))
	require.Equal(t, "juliet@jabber.org", respStanzas[0].Attribute(stravaganza.To))
	require.Equal(t, stravaganza.AvailableType, respStanzas[0].Attribute(stravaganza.Type))

	require.Equal(t, "presence", respStanzas[1].Name())
	require.Equal(t, "ortuman@jackal.im/yard", respStanzas[1].Attribute(stravaganza.From))
	require.Equal(t, "juliet@jabber.org", respStanzas[1].Attribute(stravaganza.To))
	require.Equal(t, stravaganza.AvailableType, respStanzas[1].Attribute(stravaganza.Type))

	require.Equal(t, "iq", respStanzas[2].Name())
	require.Equal(t, stravaganza.ResultType, respStanzas[2].Attribute(stravaganza.Type))

	require.Equal(t, "iq", respStanzas[3].Name())
	require.Equal(t, "ortuman@jackal.im", respStanzas[3].Attribute(stravaganza.From))
	require.Equal(t, "ortuman@jackal.im/chamber", respStanzas[3].Attribute(stravaganza.To))
	require.Equal(t, stravaganza.SetType, respStanzas[3].Attribute(stravaganza.Type))
	require.NotNil(t, respStanzas[3].ChildNamespace("unblock", blockListNamespace))

	require.Equal(t, "iq", respStanzas[4].Name())
	require.Equal(t, "ortuman@jackal.im", respStanzas[4].Attribute(stravaganza.From))
	require.Equal(t, "ortuman@jackal.im/yard", respStanzas[4].Attribute(stravaganza.To))
	require.Equal(t, stravaganza.SetType, respStanzas[4].Attribute(stravaganza.Type))
	require.NotNil(t, respStanzas[4].ChildNamespace("unblock", blockListNamespace))
}

func TestBlockList_Forbidden(t *testing.T) {
	// given
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	bl := &BlockList{
		router: routerMock,
		hk:     hook.NewHooks(),
		logger: kitlog.NewNopLogger(),
	}

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, uuid.New().String()).
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/chamber").
		WithAttribute(stravaganza.To, "noelia@jackal.im").
		WithChild(
			stravaganza.NewBuilder("blocklist").
				WithAttribute(stravaganza.Namespace, blockListNamespace).
				Build(),
		).
		BuildIQ()

	// then
	_ = bl.ProcessIQ(context.Background(), iq)

	require.Len(t, respStanzas, 1)
	require.Equal(t, stravaganza.ErrorType, respStanzas[0].Attribute(stravaganza.Type))
}

func TestBlockList_UserDeleted(t *testing.T) {
	// given
	rep := &repositoryMock{}
	rep.DeleteBlockListItemsFunc = func(ctx context.Context, username string) error {
		return nil
	}

	hk := hook.NewHooks()
	bl := &BlockList{
		rep:    rep,
		hk:     hk,
		logger: kitlog.NewNopLogger(),
	}
	// when
	_ = bl.Start(context.Background())
	defer func() { _ = bl.Stop(context.Background()) }()

	_, _ = hk.Run(context.Background(), hook.UserDeleted, &hook.ExecutionContext{
		Info: &hook.UserInfo{
			Username: "ortuman",
		},
	})

	// then
	require.Len(t, rep.DeleteBlockListItemsCalls(), 1)
}

func TestBlockList_InterceptIncomingStanza(t *testing.T) {
	// given
	routerMock := &routerMock{}
	hMock := &hostsMock{}
	rep := &repositoryMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	rep.FetchBlockListItemsFunc = func(ctx context.Context, username string) ([]*blocklistmodel.Item, error) {
		return []*blocklistmodel.Item{
			{Username: "ortuman", Jid: "jabber.org/yard"},
		}, nil
	}
	hk := hook.NewHooks()
	bl := &BlockList{
		hosts:  hMock,
		router: routerMock,
		rep:    rep,
		hk:     hk,
		logger: kitlog.NewNopLogger(),
	}

	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "juliet@jabber.org/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()

	// when
	_ = bl.Start(context.Background())
	defer func() { _ = bl.Stop(context.Background()) }()

	halted, err := hk.Run(context.Background(), hook.C2SStreamElementReceived, &hook.ExecutionContext{
		Info: &hook.C2SStreamInfo{
			Element: msg,
		},
	})

	// then
	require.True(t, halted)
	require.Nil(t, err)

	require.Len(t, respStanzas, 1)
	require.Equal(t, "ortuman@jackal.im/balcony", respStanzas[0].Attribute(stravaganza.From))
	require.Equal(t, "juliet@jabber.org/yard", respStanzas[0].Attribute(stravaganza.To))
	require.Equal(t, stravaganza.ErrorType, respStanzas[0].Attribute(stravaganza.Type))

	errEl := respStanzas[0].Child("error")
	require.NotNil(t, errEl)

	require.NotNil(t, errEl.ChildNamespace("service-unavailable", "urn:ietf:params:xml:ns:xmpp-stanzas"))
}

func TestBlockList_InterceptOutgoingStanza(t *testing.T) {
	// given
	routerMock := &routerMock{}
	hMock := &hostsMock{}
	rep := &repositoryMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	rep.FetchBlockListItemsFunc = func(ctx context.Context, username string) ([]*blocklistmodel.Item, error) {
		return []*blocklistmodel.Item{
			{Username: "ortuman", Jid: "jabber.org/yard"},
		}, nil
	}
	hk := hook.NewHooks()
	bl := &BlockList{
		hosts:  hMock,
		router: routerMock,
		rep:    rep,
		hk:     hk,
		logger: kitlog.NewNopLogger(),
	}
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "ortuman@jackal.im/balcony")
	b.WithAttribute("to", "juliet@jabber.org/yard")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()

	// when
	_ = bl.Start(context.Background())
	defer func() { _ = bl.Stop(context.Background()) }()

	halted, err := hk.Run(context.Background(), hook.C2SStreamWillRouteElement, &hook.ExecutionContext{
		Info: &hook.C2SStreamInfo{
			Element: msg,
		},
	})

	// then
	require.Nil(t, err)
	require.True(t, halted)

	require.Len(t, respStanzas, 1)
	require.Equal(t, "juliet@jabber.org/yard", respStanzas[0].Attribute(stravaganza.From))
	require.Equal(t, "ortuman@jackal.im/balcony", respStanzas[0].Attribute(stravaganza.To))
	require.Equal(t, stravaganza.ErrorType, respStanzas[0].Attribute(stravaganza.Type))

	errEl := respStanzas[0].Child("error")
	require.NotNil(t, errEl)

	require.NotNil(t, errEl.ChildNamespace("not-acceptable", "urn:ietf:params:xml:ns:xmpp-stanzas"))
}

func TestBlockList_PresenceTargets(t *testing.T) {
	// given
	rep := &repositoryMock{}
	rep.FetchRosterItemsFunc = func(ctx context.Context, username string) ([]*rostermodel.Item, error) {
		return []*rostermodel.Item{
			{Username: "ortuman", Jid: "juliet@jabber.org", Subscription: rostermodel.Both},
			{Username: "ortuman", Jid: "hamlet@jabber.org", Subscription: rostermodel.To},
			{Username: "ortuman", Jid: "hamlet@404.city", Subscription: rostermodel.Both},
			{Username: "ortuman", Jid: "macbeth@404.city", Subscription: rostermodel.Both},
			{Username: "ortuman", Jid: "witch@404.city", Subscription: rostermodel.To},
			{Username: "ortuman", Jid: "witch@jackal.im", Subscription: rostermodel.Both},
			{Username: "ortuman", Jid: "witch@jabber.net", Subscription: rostermodel.Both},
			{Username: "ortuman", Jid: "witch@jabber.org", Subscription: rostermodel.To},
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
