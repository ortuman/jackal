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

package xep0049

import (
	"context"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/ortuman/jackal/pkg/hook"
	"github.com/stretchr/testify/require"
)

func TestPrivate_GetPrivate(t *testing.T) {
	// given
	var reqNS string

	repMock := &repositoryMock{}
	repMock.FetchPrivateFunc = func(ctx context.Context, namespace, username string) (stravaganza.Element, error) {
		reqNS = namespace
		return stravaganza.NewBuilder("exodus").
			WithAttribute(stravaganza.Namespace, "exodus:prefs").
			WithChild(
				stravaganza.NewBuilder("defaultnick").
					WithText("Hamlet").
					Build(),
			).
			Build(), nil
	}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	// when
	p := &Private{
		rep:    repMock,
		router: routerMock,
		hk:     hook.NewHooks(),
		logger: kitlog.NewNopLogger(),
	}
	reqIQ, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithAttribute(stravaganza.ID, "1001").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, privateNamespace).
				WithChild(
					stravaganza.NewBuilder("exodus").
						WithAttribute(stravaganza.Namespace, "exodus:prefs").
						Build(),
				).
				Build(),
		).
		BuildIQ()

	_ = p.ProcessIQ(context.Background(), reqIQ)

	// then
	require.Len(t, respStanzas, 1)

	resIQ := respStanzas[0]
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute(stravaganza.Type))

	q := resIQ.ChildNamespace("query", privateNamespace)
	require.NotNil(t, q)
	require.Equal(t, q.ChildrenCount(), 1)

	require.Equal(t, reqNS, "exodus:prefs")
}

func TestPrivate_SetPrivate(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertPrivateFunc = func(ctx context.Context, private stravaganza.Element, namespace string, username string) error {
		return nil
	}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	// when
	p := &Private{
		rep:    repMock,
		router: routerMock,
		hk:     hook.NewHooks(),
		logger: kitlog.NewNopLogger(),
	}
	reqIQ, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithAttribute(stravaganza.ID, "1001").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, privateNamespace).
				WithChild(
					stravaganza.NewBuilder("exodus").
						WithAttribute(stravaganza.Namespace, "exodus:prefs").
						WithChild(
							stravaganza.NewBuilder("defaultnick").
								WithText("Hamlet").
								Build(),
						).
						Build(),
				).
				Build(),
		).
		BuildIQ()

	_ = p.ProcessIQ(context.Background(), reqIQ)

	// then
	require.Len(t, respStanzas, 1)

	resIQ := respStanzas[0]
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute(stravaganza.Type))
}

func TestPrivate_ForbiddenRequest(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	// when
	p := &Private{
		rep:    repMock,
		router: routerMock,
		hk:     hook.NewHooks(),
		logger: kitlog.NewNopLogger(),
	}
	reqIQ, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithAttribute(stravaganza.ID, "1001").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "noelia@jackal.im").
		WithChild(
			stravaganza.NewBuilder("query").
				WithAttribute(stravaganza.Namespace, privateNamespace).
				WithChild(
					stravaganza.NewBuilder("exodus").
						WithAttribute(stravaganza.Namespace, "exodus:prefs").
						Build(),
				).
				Build(),
		).
		BuildIQ()

	_ = p.ProcessIQ(context.Background(), reqIQ)

	// then
	require.Len(t, respStanzas, 1)

	resIQ := respStanzas[0]
	require.Equal(t, stravaganza.ErrorType, resIQ.Attribute(stravaganza.Type))

	err := resIQ.Child("error")
	require.NotNil(t, err)

	require.NotNil(t, err.Children("forbidden"))
}
