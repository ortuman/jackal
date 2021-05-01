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

package xep0280

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/ortuman/jackal/pkg/event"
	coremodel "github.com/ortuman/jackal/pkg/model/core"
	"github.com/ortuman/jackal/pkg/router"
	"github.com/ortuman/jackal/pkg/router/stream"
	"github.com/stretchr/testify/require"
)

func TestCarbons_Enable(t *testing.T) {
	// given
	stmMock := &c2sStreamMock{}

	var setK string
	var setVal bool
	stmMock.SetInfoValueFunc = func(ctx context.Context, k string, val interface{}) error {
		setK = k
		setVal = val.(bool)
		return nil
	}
	stmMock.InfoFunc = func() *coremodel.ResourceInfo {
		return &coremodel.ResourceInfo{
			M: map[string]string{},
		}
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
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}

	c := &Carbons{
		router: routerMock,
		hosts:  hMock,
		sn:     sonar.New(),
	}
	// when
	setID := uuid.New().String()

	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, setID).
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("enable").
				WithAttribute(stravaganza.Namespace, carbonsNamespace).
				Build(),
		).
		BuildIQ()

	_ = c.ProcessIQ(context.Background(), iq)

	// then
	require.Equal(t, enabledInfoKey, setK)
	require.Equal(t, true, setVal)

	require.Len(t, respStanzas, 1)

	require.Equal(t, stravaganza.IQName, respStanzas[0].Name())
	require.Equal(t, setID, respStanzas[0].Attribute(stravaganza.ID))
	require.Equal(t, stravaganza.ResultType, respStanzas[0].Attribute(stravaganza.Type))
}

func TestCarbons_Disable(t *testing.T) {
	// given
	stmMock := &c2sStreamMock{}

	var setK string
	var setVal bool
	stmMock.SetInfoValueFunc = func(ctx context.Context, k string, val interface{}) error {
		setK = k
		setVal = val.(bool)
		return nil
	}
	stmMock.InfoFunc = func() *coremodel.ResourceInfo {
		return &coremodel.ResourceInfo{
			M: map[string]string{},
		}
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
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}

	c := &Carbons{
		router: routerMock,
		hosts:  hMock,
		sn:     sonar.New(),
	}
	// when
	setID := uuid.New().String()

	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, setID).
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("disable").
				WithAttribute(stravaganza.Namespace, carbonsNamespace).
				Build(),
		).
		BuildIQ()

	_ = c.ProcessIQ(context.Background(), iq)

	// then
	require.Equal(t, enabledInfoKey, setK)
	require.Equal(t, false, setVal)

	require.Len(t, respStanzas, 1)

	require.Equal(t, stravaganza.IQName, respStanzas[0].Name())
	require.Equal(t, setID, respStanzas[0].Attribute(stravaganza.ID))
	require.Equal(t, stravaganza.ResultType, respStanzas[0].Attribute(stravaganza.Type))
}

func TestCarbons_SentCC(t *testing.T) {
	// given
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	jd0, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)

	resManagerMock := &resourceManagerMock{}
	resManagerMock.GetResourcesFunc = func(ctx context.Context, username string) ([]coremodel.Resource, error) {
		return []coremodel.Resource{
			{JID: jd0, Info: coremodel.ResourceInfo{
				M: map[string]string{enabledInfoKey: "true"},
			}},
		}, nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}

	sn := sonar.New()
	c := &Carbons{
		router: routerMock,
		resMng: resManagerMock,
		hosts:  hMock,
		sn:     sn,
	}

	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("id", "i1234")
	b.WithAttribute("from", "ortuman@jackal.im/yard")
	b.WithAttribute("to", "noelia@jabber.org/balcony")
	b.WithAttribute("type", "chat")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()

	// when
	_ = c.Start(context.Background())
	defer func() { _ = c.Stop(context.Background()) }()

	_ = sn.Post(context.Background(), sonar.NewEventBuilder(event.S2SInStreamMessageRouted).
		WithInfo(&event.S2SStreamEventInfo{
			Sender:  "jackal.im",
			Target:  "jabber.org",
			Element: msg,
		}).
		Build(),
	)

	// then
	require.Len(t, respStanzas, 1)

	routedMsg := respStanzas[0]

	require.Equal(t, stravaganza.MessageName, routedMsg.Name())
	require.Equal(t, "ortuman@jackal.im", routedMsg.Attribute(stravaganza.From))
	require.Equal(t, "ortuman@jackal.im/balcony", routedMsg.Attribute(stravaganza.To))
	require.NotNil(t, routedMsg.ChildNamespace("sent", carbonsNamespace))
}

func TestCarbons_ReceivedCC(t *testing.T) {
	// given
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	jd0, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)
	jd1, _ := jid.NewWithString("ortuman@jackal.im/hall", true)
	jd2, _ := jid.NewWithString("ortuman@jackal.im/chamber", true)

	resManagerMock := &resourceManagerMock{}
	resManagerMock.GetResourcesFunc = func(ctx context.Context, username string) ([]coremodel.Resource, error) {
		return []coremodel.Resource{
			{JID: jd0, Info: coremodel.ResourceInfo{M: map[string]string{enabledInfoKey: "true"}}},
			{JID: jd1, Info: coremodel.ResourceInfo{M: map[string]string{enabledInfoKey: "false"}}},
			{JID: jd2, Info: coremodel.ResourceInfo{M: map[string]string{enabledInfoKey: "true"}}},
		}, nil
	}

	hMock := &hostsMock{}
	hMock.IsLocalHostFunc = func(h string) bool {
		return h == "jackal.im"
	}

	sn := sonar.New()
	c := &Carbons{
		router: routerMock,
		resMng: resManagerMock,
		hosts:  hMock,
		sn:     sn,
	}

	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("id", "i1234")
	b.WithAttribute("from", "noelia@jabber.org/balcony")
	b.WithAttribute("to", "ortuman@jackal.im/chamber")
	b.WithAttribute("type", "chat")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()

	// when
	_ = c.Start(context.Background())
	defer func() { _ = c.Stop(context.Background()) }()

	_ = sn.Post(context.Background(), sonar.NewEventBuilder(event.C2SStreamMessageRouted).
		WithInfo(&event.C2SStreamEventInfo{
			Targets: []jid.JID{*jd2},
			Element: msg,
		}).
		Build(),
	)

	// then
	require.Len(t, respStanzas, 1)

	routedMsg := respStanzas[0]

	require.Equal(t, stravaganza.MessageName, routedMsg.Name())
	require.Equal(t, "ortuman@jackal.im", routedMsg.Attribute(stravaganza.From))
	require.Equal(t, "ortuman@jackal.im/balcony", routedMsg.Attribute(stravaganza.To))
	require.NotNil(t, routedMsg.ChildNamespace("received", carbonsNamespace))
}

func TestCarbons_InterceptStanza(t *testing.T) {
	// given
	c := &Carbons{
		sn: sonar.New(),
	}

	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	b.WithChild(
		stravaganza.NewBuilder("private").
			WithAttribute(stravaganza.Namespace, carbonsNamespace).
			Build(),
	)
	msg, _ := b.BuildMessage()

	// when
	tst, err := c.InterceptStanza(context.Background(), msg, 0)

	// then
	require.Nil(t, err)
	require.Nil(t, tst.ChildNamespace("private", carbonsNamespace))
}
