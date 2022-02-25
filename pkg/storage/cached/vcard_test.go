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

package cachedrepository

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza"
	"github.com/stretchr/testify/require"
)

func TestCachedVCardRep_UpsertVCard(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertVCardFunc = func(ctx context.Context, vcard stravaganza.Element, username string) error {
		return nil
	}

	// when
	rep := cachedVCardRep{
		c:   cacheMock,
		rep: repMock,
	}
	vCard := stravaganza.NewBuilder("vCard").Build()

	err := rep.UpsertVCard(context.Background(), vCard, "u1")

	// then
	require.NoError(t, err)
	require.Equal(t, vCardNS("u1"), cacheNS)
	require.Equal(t, vCardKey, cacheKey)
	require.Len(t, repMock.UpsertVCardCalls(), 1)
}

func TestCachedVCardRep_DeleteVCard(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteVCardFunc = func(ctx context.Context, username string) error {
		return nil
	}

	// when
	rep := cachedVCardRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteVCard(context.Background(), "v1")

	// then
	require.NoError(t, err)
	require.Equal(t, vCardNS("v1"), cacheNS)
	require.Equal(t, vCardKey, cacheKey)
	require.Len(t, repMock.DeleteVCardCalls(), 1)
}

func TestCachedVCardRep_FetchVCard(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchVCardFunc = func(ctx context.Context, username string) (stravaganza.Element, error) {
		vCard := stravaganza.NewBuilder("vCard").Build()
		return vCard, nil
	}

	// when
	rep := cachedVCardRep{
		c:   cacheMock,
		rep: repMock,
	}
	vCard, err := rep.FetchVCard(context.Background(), "u1")

	// then
	require.NotNil(t, vCard)
	require.NoError(t, err)

	require.Equal(t, "vCard", vCard.Name())

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchVCardCalls(), 1)
}
