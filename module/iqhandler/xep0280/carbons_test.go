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
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/sonar"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/router/stream"
	"github.com/stretchr/testify/require"
)

func TestCarbons_Enable(t *testing.T) {
	// given
	stmMock := &c2sStreamMock{}

	var setK, setVal string
	stmMock.SetValueFunc = func(ctx context.Context, k string, val string) error {
		setK = k
		setVal = val
		return nil
	}
	stmMock.ValueFunc = func(cKey string) string {
		return ""
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
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) error {
		respStanzas = append(respStanzas, stanza)
		return nil
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
		BuildIQ(false)

	_ = c.ProcessIQ(context.Background(), iq)

	// then
	require.Equal(t, carbonsEnabledCtxKey, setK)
	require.Equal(t, strconv.FormatBool(true), setVal)

	require.Len(t, respStanzas, 1)

	require.Equal(t, stravaganza.IQName, respStanzas[0].Name())
	require.Equal(t, setID, respStanzas[0].Attribute(stravaganza.ID))
	require.Equal(t, stravaganza.ResultType, respStanzas[0].Attribute(stravaganza.Type))
}

func TestCarbons_Disable(t *testing.T) {
	// given
	stmMock := &c2sStreamMock{}

	var setK, setVal string
	stmMock.SetValueFunc = func(ctx context.Context, k string, val string) error {
		setK = k
		setVal = val
		return nil
	}
	stmMock.ValueFunc = func(cKey string) string {
		return ""
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
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) error {
		respStanzas = append(respStanzas, stanza)
		return nil
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
		BuildIQ(false)

	_ = c.ProcessIQ(context.Background(), iq)

	// then
	require.Equal(t, carbonsEnabledCtxKey, setK)
	require.Equal(t, strconv.FormatBool(false), setVal)

	require.Len(t, respStanzas, 1)

	require.Equal(t, stravaganza.IQName, respStanzas[0].Name())
	require.Equal(t, setID, respStanzas[0].Attribute(stravaganza.ID))
	require.Equal(t, stravaganza.ResultType, respStanzas[0].Attribute(stravaganza.Type))
}

func TestCarbons_SentCC(t *testing.T) {
	// given
	// when
	// then
}

func TestCarbons_ReceivedCC(t *testing.T) {
	// given
	// when
	// then
}
