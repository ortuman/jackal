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

package xep0030

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/component"
	"github.com/ortuman/jackal/model"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module"
	"github.com/stretchr/testify/require"
)

func TestDisco_GetServerInfo(t *testing.T) {
	// given
	modMock := &moduleMock{}
	modMock.ServerFeaturesFunc = func(_ context.Context) ([]string, error) {
		return []string{"https://jackal.im#feature-1", "https://jackal.im#feature-2"}, nil
	}

	routerMock := &routerMock{}
	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	d := &Disco{
		router: routerMock,
	}
	d.SetModules([]module.Module{modMock, d})

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, discoInfoNamespace).
				Build(),
		).
		BuildIQ(false)
	_ = d.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 1)

	resIQ, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute("type"))

	query := resIQ.ChildNamespace("query", discoInfoNamespace)
	require.NotNil(t, query)

	identity := query.Child("identity")
	require.NotNil(t, identity)
	require.Equal(t, "server", identity.Attribute("category"))
	require.Equal(t, "jackal", identity.Attribute("name"))

	features := query.Children("feature")
	require.Len(t, features, 4)
}

func TestDisco_GetServerItems(t *testing.T) {
	// given
	routerMock := &routerMock{}
	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	compMock := &componentMock{}
	compMock.NameFunc = func() string {
		return "comp-1"
	}
	compMock.HostFunc = func() string {
		return "host.jackal.im"
	}
	compsMock := &componentsMock{}
	compsMock.AllComponentsFunc = func() []component.Component {
		return []component.Component{compMock}
	}
	d := &Disco{
		router:     routerMock,
		components: compsMock,
	}
	d.SetModules(nil)

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, discoItemsNamespace).
				Build(),
		).
		BuildIQ(false)
	_ = d.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 1)

	resIQ, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute("type"))

	query := resIQ.ChildNamespace("query", discoItemsNamespace)
	require.NotNil(t, query)

	items := query.Children("item")
	require.Len(t, items, 1)

	require.Equal(t, "comp-1", items[0].Attribute("name"))
	require.Equal(t, "host.jackal.im", items[0].Attribute("jid"))
}

func TestDisco_GetAccountInfo(t *testing.T) {
	// given
	modMock := &moduleMock{}
	modMock.AccountFeaturesFunc = func(_ context.Context) ([]string, error) {
		return []string{"https://jackal.im#feature-1", "https://jackal.im#feature-2"}, nil
	}

	routerMock := &routerMock{}
	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	repMock := &rosterRepositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		return &rostermodel.Item{
			Username:     "ortuman",
			JID:          "noelia@jackal.im",
			Subscription: rostermodel.To,
		}, nil
	}

	d := &Disco{
		router: routerMock,
		rosRep: repMock,
	}
	d.SetModules([]module.Module{modMock, d})

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "noelia@jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, discoInfoNamespace).
				Build(),
		).
		BuildIQ(false)
	_ = d.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 1)

	resIQ, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute("type"))

	query := resIQ.ChildNamespace("query", discoInfoNamespace)
	require.NotNil(t, query)

	identity := query.Child("identity")
	require.NotNil(t, identity)
	require.Equal(t, "account", identity.Attribute("category"))

	features := query.Children("feature")
	require.Len(t, features, 4)
}

func TestDisco_GetAccountItems(t *testing.T) {
	// given
	routerMock := &routerMock{}
	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}
	repMock := &rosterRepositoryMock{}
	repMock.FetchRosterItemFunc = func(ctx context.Context, username string, jid string) (*rostermodel.Item, error) {
		return &rostermodel.Item{
			Username:     "ortuman",
			JID:          "noelia@jackal.im",
			Subscription: rostermodel.To,
		}, nil
	}
	jd0, _ := jid.NewWithString("noelia@jackal.im/chamber", true)
	resMng := &resourceManagerMock{}
	resMng.GetResourcesFunc = func(ctx context.Context, username string) ([]model.Resource, error) {
		return []model.Resource{
			{
				InstanceID: "inst-1",
				JID:        jd0,
			},
		}, nil
	}
	d := &Disco{
		router: routerMock,
		rosRep: repMock,
		resMng: resMng,
	}
	d.SetModules(nil)

	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "noelia@jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, discoItemsNamespace).
				Build(),
		).
		BuildIQ(false)
	_ = d.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 1)

	resIQ, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute("type"))

	query := resIQ.ChildNamespace("query", discoItemsNamespace)
	require.NotNil(t, query)

	items := query.Children("item")
	require.Len(t, items, 1)

	require.Equal(t, "noelia@jackal.im/chamber", items[0].Attribute("jid"))
}
