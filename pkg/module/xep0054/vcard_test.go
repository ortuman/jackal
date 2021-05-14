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

package xep0054

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/pkg/module/hook"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/jackal-xmpp/stravaganza/v2/jid"
	"github.com/stretchr/testify/require"
)

func TestVCard_Features(t *testing.T) {
	// given
	v := &VCard{}

	// when
	srvFeatures, _ := v.ServerFeatures(context.Background())
	accFeatures, _ := v.AccountFeatures(context.Background())

	// then
	require.Equal(t, []string{vCardNamespace}, accFeatures)
	require.Equal(t, []string{vCardNamespace}, srvFeatures)
}

func TestVCard_GetVCard(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchVCardFunc = func(ctx context.Context, username string) (stravaganza.Element, error) {
		return stravaganza.NewBuilder("vCard").
			WithAttribute(stravaganza.Namespace, vCardNamespace).
			WithChild(
				stravaganza.NewBuilder("FN").
					WithText("Noelia").
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

	v := &VCard{
		rep:    repMock,
		router: routerMock,
		hk:     hook.NewHooks(),
	}
	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.GetType).
		WithChild(
			stravaganza.NewBuilder("vCard").
				WithAttribute(stravaganza.Namespace, vCardNamespace).
				Build(),
		).
		BuildIQ()
	_ = v.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 1)

	resIQ, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute("type"))

	vCard := resIQ.ChildNamespace("vCard", vCardNamespace)
	require.NotNil(t, vCard)

	fn := vCard.Child("FN")
	require.NotNil(t, fn)
	require.Equal(t, "Noelia", fn.Text())
}

func TestVCard_SetVCard(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertVCardFunc = func(ctx context.Context, vCard stravaganza.Element, username string) error {
		return nil
	}
	routerMock := &routerMock{}

	var respStanzas []stravaganza.Stanza
	routerMock.RouteFunc = func(ctx context.Context, stanza stravaganza.Stanza) ([]jid.JID, error) {
		respStanzas = append(respStanzas, stanza)
		return nil, nil
	}

	v := &VCard{
		rep:    repMock,
		router: routerMock,
		hk:     hook.NewHooks(),
	}
	// when
	iq, _ := stravaganza.NewIQBuilder().
		WithAttribute(stravaganza.ID, "id1234").
		WithAttribute(stravaganza.From, "ortuman@jackal.im/yard").
		WithAttribute(stravaganza.To, "ortuman@jackal.im").
		WithAttribute(stravaganza.Type, stravaganza.SetType).
		WithChild(
			stravaganza.NewBuilder("vCard").
				WithAttribute(stravaganza.Namespace, vCardNamespace).
				WithChild(
					stravaganza.NewBuilder("FN").
						WithText("Noelia").
						Build(),
				).
				Build(),
		).
		BuildIQ()
	_ = v.ProcessIQ(context.Background(), iq)

	// then
	require.Len(t, respStanzas, 1)

	resIQ, ok := respStanzas[0].(*stravaganza.IQ)
	require.True(t, ok)
	require.Equal(t, stravaganza.ResultType, resIQ.Attribute("type"))
	require.Len(t, resIQ.AllChildren(), 0)
}
