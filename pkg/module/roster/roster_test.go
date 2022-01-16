// Copyright 2020 The jackal Authors
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

package roster

import (
	"context"
	"sync"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/hook"
	c2smodel "github.com/ortuman/jackal/pkg/model/c2s"
	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/ortuman/jackal/pkg/storage/repository"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
	"github.com/stretchr/testify/require"
)

func TestRoster_SendRoster(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 1, nil
	}
	repMock.FetchRosterItemsFunc = func(ctx context.Context, username string) ([]*rostermodel.Item, error) {
		return []*rostermodel.Item{
			{Username: "ortuman", Jid: "noelia@jackal.im", Groups: []string{"VIP"}},
			{Username: "ortuman", Jid: "shakespeare@jackal.im", Groups: []string{"Buddies"}},
		}, nil
	}

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
	routerMock := &routerMock{}

	var respStanza stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanza = stanza
		return nil, nil
	}
	routerMock.C2SFunc = func() router.C2SRouter {
		return c2sRouterMock
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	resMngMock := &resourceManagerMock{}

	r := &Roster{
		rep:    repMock,
		resMng: resMngMock,
		router: routerMock,
		hosts:  hMock,
		hk:     hook.NewHooks(),
		logger: kitlog.NewNopLogger(),
	}
	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, rosterNamespace).
				Build(),
		).
		BuildIQ()
	_ = r.ProcessIQ(context.Background(), iq)

	// then
	respIQ, ok := respStanza.(*stravaganza.IQ)
	require.True(t, ok)

	query := respIQ.ChildNamespace("query", rosterNamespace)
	require.NotNil(t, query)

	items := query.Children("item")
	require.Len(t, items, 2)

	require.Equal(t, rosterRequestedCtxKey, setK)
	require.Equal(t, true, setVal)

	require.Len(t, stmMock.SetInfoValueCalls(), 1)
}

func TestRoster_UpdateItem(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		return nil, nil
	}
	txMock := &txMock{}
	txMock.TouchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 1, nil
	}
	txMock.UpsertRosterItemFunc = func(ctx context.Context, ri *rostermodel.Item) error {
		return nil
	}
	repMock.InTransactionFunc = func(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
		return f(ctx, txMock)
	}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	routerMock.C2SFunc = func() router.C2SRouter {
		return nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	resMngMock := &resourceManagerMock{}

	jd0, _ := jid.New("ortuman", "jackal.im", "yard", true)
	jd1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	resMngMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.ResourceDesc, error) {
		return []c2smodel.ResourceDesc{
			c2smodel.NewResourceDesc("i0", jd0, nil, c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"})),
			c2smodel.NewResourceDesc("i1", jd1, nil, c2smodel.NewInfoMap()),
		}, nil
	}

	r := &Roster{
		rep:    repMock,
		resMng: resMngMock,
		router: routerMock,
		hosts:  hMock,
		hk:     hook.NewHooks(),
		logger: kitlog.NewNopLogger(),
	}
	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, rosterNamespace).
				WithChild(
					stravaganza.NewBuilder("item").
						WithAttribute("name", "Buddy").
						WithAttribute("jid", "hamlet@jackal.im").
						WithAttribute("subscription", "none").
						Build(),
				).
				Build(),
		).
		BuildIQ()
	_ = r.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 2)

	pushIQ, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)

	resIQ, ok := respStanzas[1].(*stravaganza.IQ)
	require.True(t, ok)

	require.Equal(t, stravaganza.SetType, pushIQ.Attribute("type"))

	query := pushIQ.ChildNamespace("query", rosterNamespace)
	require.NotNil(t, query)

	item := query.Child("item")

	require.NotNil(t, item)
	require.Equal(t, "hamlet@jackal.im", item.Attribute("jid"))
	require.Equal(t, rostermodel.None, item.Attribute("subscription"))

	require.Equal(t, "id1234", resIQ.Attribute("id"))
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute("type"))
}

func TestRoster_RemoveItem(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		switch {
		case username == "ortuman" && jid == "hamlet@jackal.im":
			return &rostermodel.Item{
				Username:     "ortuman",
				Jid:          "hamlet@jackal.im",
				Subscription: "both",
			}, nil
		}
		return nil, nil
	}
	repMock.FetchRosterNotificationFunc = func(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
		return nil, nil
	}
	txMock := &txMock{}
	txMock.TouchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 1, nil
	}
	txMock.DeleteRosterItemFunc = func(ctx context.Context, username string, jid string) error {
		return nil
	}
	repMock.InTransactionFunc = func(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
		return f(ctx, txMock)
	}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	routerMock.C2SFunc = func() router.C2SRouter {
		return nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	jd0, _ := jid.New("ortuman", "jackal.im", "yard", true)
	jd1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	resMngMock := &resourceManagerMock{}
	resMngMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.ResourceDesc, error) {
		return []c2smodel.ResourceDesc{
			c2smodel.NewResourceDesc("i0", jd0, nil, c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"})),
			c2smodel.NewResourceDesc("i1", jd1, nil, c2smodel.NewInfoMap()),
		}, nil
	}

	r := &Roster{
		rep:    repMock,
		resMng: resMngMock,
		router: routerMock,
		hosts:  hMock,
		hk:     hook.NewHooks(),
		logger: kitlog.NewNopLogger(),
	}
	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, rosterNamespace).
				WithChild(
					stravaganza.NewBuilder("item").
						WithAttribute("name", "Buddy").
						WithAttribute("jid", "hamlet@jackal.im").
						WithAttribute("subscription", "remove").
						Build(),
				).
				Build(),
		).
		BuildIQ()
	_ = r.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 6)

	pushIQ, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	query := pushIQ.ChildNamespace("query", rosterNamespace)
	require.NotNil(t, query)

	item := query.Child("item")
	require.NotNil(t, item)
	require.Equal(t, "hamlet@jackal.im", item.Attribute("jid"))
	require.Equal(t, rostermodel.Remove, item.Attribute("subscription"))

	unsubscribePr, ok := respStanzas[1].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, stravaganza.UnsubscribeType, unsubscribePr.Attribute("type"))

	unsubscribedPr, ok := respStanzas[2].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, stravaganza.UnsubscribedType, unsubscribedPr.Attribute("type"))

	unavailablePr0, ok := respStanzas[3].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, stravaganza.UnavailableType, unavailablePr0.Attribute("type"))

	unavailablePr1, ok := respStanzas[4].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, stravaganza.UnavailableType, unavailablePr1.Attribute("type"))

	resIQ, ok := respStanzas[5].(*stravaganza.IQ)
	require.True(t, ok)
	require.Equal(t, stravaganza.SetType, pushIQ.Attribute("type"))
	require.Equal(t, "id1234", resIQ.Attribute("id"))
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute("type"))
}

func TestRoster_Subscribe(t *testing.T) {
	// given
	var mtx sync.RWMutex

	repMock := &repositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		return nil, nil
	}
	txMock := &txMock{}
	txMock.TouchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 2, nil
	}
	txMock.UpsertRosterItemFunc = func(ctx context.Context, ri *rostermodel.Item) error {
		return nil
	}
	repMock.InTransactionFunc = func(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
		return f(ctx, txMock)
	}
	repMock.UpsertRosterNotificationFunc = func(ctx context.Context, rn *rostermodel.Notification) error {
		return nil
	}

	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		mtx.Lock()
		defer mtx.Unlock()
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	jd0, _ := jid.New("ortuman", "jackal.im", "yard", true)
	jd1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	resMngMock := &resourceManagerMock{}
	resMngMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.ResourceDesc, error) {
		return []c2smodel.ResourceDesc{
			c2smodel.NewResourceDesc("i0", jd0, nil, c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"})),
			c2smodel.NewResourceDesc("i1", jd1, nil, c2smodel.NewInfoMap()),
		}, nil
	}

	hk := hook.NewHooks()
	r := &Roster{
		rep:    repMock,
		resMng: resMngMock,
		router: routerMock,
		hosts:  hMock,
		hk:     hk,
		logger: kitlog.NewNopLogger(),
	}
	// when
	fromJID, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)
	toJID, _ := jid.NewWithString("noelia@jackal.im", true)

	pr := xmpputil.MakePresence(fromJID, toJID, stravaganza.SubscribeType, nil)

	_ = r.Start(context.Background())
	_, _ = hk.Run(context.Background(), hook.C2SStreamPresenceReceived, &hook.ExecutionContext{
		Info: &hook.C2SStreamInfo{Element: pr},
	})

	// then
	mtx.RLock()
	defer mtx.RUnlock()

	require.Len(t, respStanzas, 2)

	pushIQ, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.NotNil(t, pushIQ.ChildNamespace("query", rosterNamespace))

	subscribePr, ok := respStanzas[1].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, stravaganza.SubscribeType, subscribePr.Attribute("type"))
}

func TestRoster_Subscribed(t *testing.T) {
	// given
	var mtx sync.RWMutex

	repMock := &repositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		switch {
		case username == "ortuman" && jid == "noelia@jackal.im":
			return &rostermodel.Item{
				Username:     "ortuman",
				Jid:          "noelia@jackal.im",
				Subscription: rostermodel.From,
			}, nil
		case username == "noelia" && jid == "ortuman@jackal.im":
			return &rostermodel.Item{
				Username:     "noelia",
				Jid:          "ortuman@jackal.im",
				Subscription: rostermodel.To,
			}, nil
		}
		return nil, nil
	}
	txMock := &txMock{}
	txMock.TouchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 2, nil
	}
	txMock.UpsertRosterItemFunc = func(ctx context.Context, ri *rostermodel.Item) error {
		return nil
	}
	repMock.InTransactionFunc = func(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
		return f(ctx, txMock)
	}
	repMock.FetchRosterNotificationFunc = func(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
		return nil, nil
	}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		mtx.Lock()
		defer mtx.Unlock()
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	jd0, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	jd1, _ := jid.New("noelia", "jackal.im", "yard", true)

	resMngMock := &resourceManagerMock{}
	resMngMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.ResourceDesc, error) {
		switch {
		case username == "ortuman":
			return []c2smodel.ResourceDesc{
				c2smodel.NewResourceDesc("i0", jd0, nil, c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"})),
			}, nil
		case username == "noelia":
			return []c2smodel.ResourceDesc{
				c2smodel.NewResourceDesc("i1", jd1, nil, c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"})),
			}, nil
		}
		return nil, nil
	}

	hk := hook.NewHooks()
	r := &Roster{
		rep:    repMock,
		resMng: resMngMock,
		router: routerMock,
		hosts:  hMock,
		hk:     hk,
		logger: kitlog.NewNopLogger(),
	}
	// when
	fromJID, _ := jid.NewWithString("noelia@jackal.im/yard", true)
	toJID, _ := jid.NewWithString("ortuman@jackal.im", true)

	pr := xmpputil.MakePresence(fromJID, toJID, stravaganza.SubscribedType, nil)

	_ = r.Start(context.Background())
	_, _ = hk.Run(context.Background(), hook.C2SStreamPresenceReceived, &hook.ExecutionContext{
		Info: &hook.C2SStreamInfo{Element: pr},
	})

	// then
	mtx.RLock()
	defer mtx.RUnlock()

	require.Len(t, respStanzas, 4)

	push0, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.NotNil(t, push0.ChildNamespace("query", rosterNamespace))

	push1, ok := respStanzas[1].(*stravaganza.IQ)
	require.True(t, ok)
	require.NotNil(t, push1.ChildNamespace("query", rosterNamespace))

	subscribedPr, ok := respStanzas[2].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, "ortuman@jackal.im", subscribedPr.Attribute("to"))
	require.Equal(t, stravaganza.SubscribedType, subscribedPr.Attribute("type"))

	availPr, ok := respStanzas[3].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, "noelia@jackal.im/yard", availPr.Attribute("from"))
	require.Equal(t, "ortuman@jackal.im", availPr.Attribute("to"))
	require.Equal(t, stravaganza.AvailableType, availPr.Attribute("type"))
}

func TestRoster_Unsubscribe(t *testing.T) {
	// given
	var mtx sync.RWMutex

	repMock := &repositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		switch {
		case username == "ortuman" && jid == "noelia@jackal.im":
			return &rostermodel.Item{
				Username:     "ortuman",
				Jid:          "noelia@jackal.im",
				Subscription: rostermodel.Both,
			}, nil
		case username == "noelia" && jid == "ortuman@jackal.im":
			return &rostermodel.Item{
				Username:     "noelia",
				Jid:          "ortuman@jackal.im",
				Subscription: rostermodel.Both,
			}, nil
		}
		return nil, nil
	}
	txMock := &txMock{}
	txMock.TouchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 2, nil
	}
	txMock.UpsertRosterItemFunc = func(ctx context.Context, ri *rostermodel.Item) error {
		return nil
	}
	repMock.InTransactionFunc = func(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
		return f(ctx, txMock)
	}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		mtx.Lock()
		defer mtx.Unlock()
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	jd0, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	jd1, _ := jid.New("noelia", "jackal.im", "yard", true)

	resMngMock := &resourceManagerMock{}
	resMngMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.ResourceDesc, error) {
		switch {
		case username == "ortuman":
			return []c2smodel.ResourceDesc{
				c2smodel.NewResourceDesc("i1", jd0, nil, c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"})),
			}, nil
		case username == "noelia":
			return []c2smodel.ResourceDesc{
				c2smodel.NewResourceDesc("i1", jd1, nil, c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"})),
			}, nil
		}
		return nil, nil
	}

	hk := hook.NewHooks()
	r := &Roster{
		rep:    repMock,
		resMng: resMngMock,
		router: routerMock,
		hosts:  hMock,
		hk:     hk,
		logger: kitlog.NewNopLogger(),
	}
	// when
	fromJID, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)
	toJID, _ := jid.NewWithString("noelia@jackal.im", true)

	pr := xmpputil.MakePresence(fromJID, toJID, stravaganza.UnsubscribeType, nil)

	_ = r.Start(context.Background())
	_, _ = hk.Run(context.Background(), hook.C2SStreamPresenceReceived, &hook.ExecutionContext{
		Info: &hook.C2SStreamInfo{Element: pr},
	})

	// then
	mtx.RLock()
	defer mtx.RUnlock()

	require.Len(t, respStanzas, 4)

	push0, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.NotNil(t, push0.ChildNamespace("query", rosterNamespace))

	push1, ok := respStanzas[1].(*stravaganza.IQ)
	require.True(t, ok)
	require.NotNil(t, push1.ChildNamespace("query", rosterNamespace))

	unsubscribePr, ok := respStanzas[2].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, "noelia@jackal.im", unsubscribePr.Attribute("to"))
	require.Equal(t, stravaganza.UnsubscribeType, unsubscribePr.Attribute("type"))

	unavailPr, ok := respStanzas[3].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, "noelia@jackal.im/yard", unavailPr.Attribute("from"))
	require.Equal(t, "ortuman@jackal.im", unavailPr.Attribute("to"))
	require.Equal(t, stravaganza.UnavailableType, unavailPr.Attribute("type"))
}

func TestRoster_Unsubscribed(t *testing.T) {
	// given
	var mtx sync.RWMutex

	repMock := &repositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		switch {
		case username == "ortuman" && jid == "noelia@jackal.im":
			return &rostermodel.Item{
				Username:     "ortuman",
				Jid:          "noelia@jackal.im",
				Subscription: rostermodel.To,
			}, nil
		case username == "noelia" && jid == "ortuman@jackal.im":
			return &rostermodel.Item{
				Username:     "noelia",
				Jid:          "ortuman@jackal.im",
				Subscription: rostermodel.From,
			}, nil
		}
		return nil, nil
	}
	txMock := &txMock{}
	txMock.TouchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 2, nil
	}
	txMock.UpsertRosterItemFunc = func(ctx context.Context, ri *rostermodel.Item) error {
		return nil
	}
	repMock.InTransactionFunc = func(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
		return f(ctx, txMock)
	}
	repMock.FetchRosterNotificationFunc = func(ctx context.Context, contact string, jid string) (*rostermodel.Notification, error) {
		return nil, nil
	}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		mtx.Lock()
		defer mtx.Unlock()
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	jd0, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	jd1, _ := jid.New("noelia", "jackal.im", "yard", true)

	resMngMock := &resourceManagerMock{}
	resMngMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.ResourceDesc, error) {
		switch {
		case username == "ortuman":
			return []c2smodel.ResourceDesc{
				c2smodel.NewResourceDesc("i1", jd0, nil, c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"})),
			}, nil
		case username == "noelia":
			return []c2smodel.ResourceDesc{
				c2smodel.NewResourceDesc("i1", jd1, nil, c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"})),
			}, nil
		}
		return nil, nil
	}

	hk := hook.NewHooks()
	r := &Roster{
		rep:    repMock,
		resMng: resMngMock,
		router: routerMock,
		hosts:  hMock,
		hk:     hk,
		logger: kitlog.NewNopLogger(),
	}
	// when
	fromJID, _ := jid.NewWithString("noelia@jackal.im/yard", true)
	toJID, _ := jid.NewWithString("ortuman@jackal.im", true)

	pr := xmpputil.MakePresence(fromJID, toJID, stravaganza.UnsubscribedType, nil)

	_ = r.Start(context.Background())
	_, _ = hk.Run(context.Background(), hook.C2SStreamPresenceReceived, &hook.ExecutionContext{
		Info: &hook.C2SStreamInfo{Element: pr},
	})

	// then
	mtx.RLock()
	defer mtx.RUnlock()

	require.Len(t, respStanzas, 4)

	push0, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.NotNil(t, push0.ChildNamespace("query", rosterNamespace))

	push1, ok := respStanzas[1].(*stravaganza.IQ)
	require.True(t, ok)
	require.NotNil(t, push1.ChildNamespace("query", rosterNamespace))

	unsubscribedPr, ok := respStanzas[2].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, "noelia@jackal.im", unsubscribedPr.Attribute("from"))
	require.Equal(t, "ortuman@jackal.im", unsubscribedPr.Attribute("to"))
	require.Equal(t, stravaganza.UnsubscribedType, unsubscribedPr.Attribute("type"))

	unavailPr, ok := respStanzas[3].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, "noelia@jackal.im/yard", unavailPr.Attribute("from"))
	require.Equal(t, "ortuman@jackal.im", unavailPr.Attribute("to"))
	require.Equal(t, stravaganza.UnavailableType, unavailPr.Attribute("type"))
}

func TestRoster_Probe(t *testing.T) {
	// given
	var mtx sync.RWMutex

	repMock := &repositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		switch {
		case username == "ortuman":
			return &rostermodel.Item{
				Username:     "ortuman",
				Jid:          "noelia@jackal.im",
				Subscription: rostermodel.Both,
			}, nil
		}
		return nil, nil
	}

	routerMock := &routerMock{}
	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		mtx.Lock()
		defer mtx.Unlock()
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	jd0, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	resMngMock := &resourceManagerMock{}

	resMngMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.ResourceDesc, error) {
		switch {
		case username == "ortuman":
			return []c2smodel.ResourceDesc{
				c2smodel.NewResourceDesc(
					"i1",
					jd0,
					xmpputil.MakePresence(jd0.ToBareJID(), jd0.ToBareJID(), stravaganza.AvailableType, nil),
					c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"}),
				),
			}, nil
		}
		return nil, nil
	}

	hk := hook.NewHooks()
	r := &Roster{
		rep:    repMock,
		resMng: resMngMock,
		router: routerMock,
		hosts:  hMock,
		hk:     hk,
		logger: kitlog.NewNopLogger(),
	}
	// when
	fromJID, _ := jid.NewWithString("noelia@jackal.im/yard", true)
	toJID, _ := jid.NewWithString("ortuman@jackal.im", true)

	pr := xmpputil.MakePresence(fromJID, toJID, stravaganza.ProbeType, nil)

	_ = r.Start(context.Background())
	_, _ = hk.Run(context.Background(), hook.C2SStreamPresenceReceived, &hook.ExecutionContext{
		Info: &hook.C2SStreamInfo{Element: pr},
	})

	// then
	mtx.RLock()
	defer mtx.RUnlock()

	require.Len(t, respStanzas, 1)

	availPr, ok := respStanzas[0].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, "ortuman@jackal.im/balcony", availPr.Attribute("from"))
	require.Equal(t, "noelia@jackal.im", availPr.Attribute("to"))
	require.Equal(t, stravaganza.AvailableType, availPr.Attribute("type"))
}

func TestRoster_Available(t *testing.T) {
	// given
	var mtx sync.RWMutex

	repMock := &repositoryMock{}
	repMock.FetchRosterItemsFunc = func(ctx context.Context, username string) ([]*rostermodel.Item, error) {
		switch {
		case username == "ortuman":
			return []*rostermodel.Item{
				{
					Username:     "ortuman",
					Jid:          "noelia@jackal.im",
					Subscription: rostermodel.Both,
				},
			}, nil
		}
		return nil, nil
	}
	repMock.FetchRosterNotificationsFunc = func(ctx context.Context, contact string) ([]*rostermodel.Notification, error) {
		return nil, nil
	}

	stmMock := &c2sStreamMock{}

	var setK string
	var setVal interface{}
	stmMock.SetInfoValueFunc = func(ctx context.Context, k string, val interface{}) error {
		mtx.Lock()
		defer mtx.Unlock()
		setK = k
		setVal = val
		return nil
	}
	stmMock.InfoFunc = func() c2smodel.Info {
		return c2smodel.NewInfoMap()
	}
	c2sRouterMock := &c2sRouterMock{}
	c2sRouterMock.LocalStreamFunc = func(username string, resource string) stream.C2S {
		return stmMock
	}

	routerMock := &routerMock{}
	routerMock.C2SFunc = func() router.C2SRouter {
		return c2sRouterMock
	}
	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		mtx.Lock()
		defer mtx.Unlock()
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}
	jd1, _ := jid.New("noelia", "jackal.im", "yard", true)

	resMngMock := &resourceManagerMock{}

	resMngMock.GetResourcesFunc = func(ctx context.Context, username string) ([]c2smodel.ResourceDesc, error) {
		switch {
		case username == "noelia":
			return []c2smodel.ResourceDesc{
				c2smodel.NewResourceDesc(
					"i1",
					jd1,
					xmpputil.MakePresence(jd1.ToBareJID(), jd1.ToBareJID(), stravaganza.AvailableType, nil),
					c2smodel.NewInfoMapFromMap(map[string]string{rosterRequestedCtxKey: "true"}),
				),
			}, nil
		}
		return nil, nil
	}

	hk := hook.NewHooks()
	r := &Roster{
		rep:    repMock,
		resMng: resMngMock,
		router: routerMock,
		hosts:  hMock,
		hk:     hk,
		logger: kitlog.NewNopLogger(),
	}
	// when
	fromJID, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)
	toJID, _ := jid.NewWithString("ortuman@jackal.im", true)

	pr := xmpputil.MakePresence(fromJID, toJID, stravaganza.AvailableType, nil)

	_ = r.Start(context.Background())
	_, _ = hk.Run(context.Background(), hook.C2SStreamPresenceReceived, &hook.ExecutionContext{
		Info: &hook.C2SStreamInfo{Element: pr},
	})

	// then
	mtx.RLock()
	defer mtx.RUnlock()

	require.Equal(t, rosterDidGoAvailableCtxKey, setK)
	require.Equal(t, true, setVal)

	require.Len(t, respStanzas, 2)

	availPr0, ok := respStanzas[0].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, "noelia@jackal.im/yard", availPr0.Attribute("from"))
	require.Equal(t, "ortuman@jackal.im/balcony", availPr0.Attribute("to"))
	require.Equal(t, stravaganza.AvailableType, availPr0.Attribute("type"))

	availPr1, ok := respStanzas[1].(*stravaganza.Presence)
	require.True(t, ok)
	require.Equal(t, "ortuman@jackal.im/balcony", availPr1.Attribute("from"))
	require.Equal(t, "noelia@jackal.im", availPr1.Attribute("to"))
	require.Equal(t, stravaganza.AvailableType, availPr1.Attribute("type"))
}
