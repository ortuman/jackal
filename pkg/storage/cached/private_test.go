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

package cachedrepository

import (
	"context"
	"testing"

	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/stretchr/testify/require"
)

func TestCachedPrivateRep_UpsertPrivate(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertPrivateFunc = func(ctx context.Context, private stravaganza.Element, namespace string, username string) error {
		return nil
	}

	// when
	rep := cachedPrivateRep{
		c:   cacheMock,
		rep: repMock,
	}
	prv := stravaganza.NewBuilder("prv").Build()

	err := rep.UpsertPrivate(context.Background(), prv, "n0", "u1")

	// then
	require.NoError(t, err)
	require.Equal(t, privateNS("u1"), cacheNS)
	require.Equal(t, "n0", cacheKey)
	require.Len(t, repMock.UpsertPrivateCalls(), 1)
}

func TestCachedPrivateRep_DeletePrivates(t *testing.T) {
	// given
	var cacheNS string

	cacheMock := &cacheMock{}
	cacheMock.DelNSFunc = func(ctx context.Context, ns string) error {
		cacheNS = ns
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeletePrivatesFunc = func(ctx context.Context, username string) error {
		return nil
	}

	// when
	rep := cachedPrivateRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeletePrivates(context.Background(), "u1")

	// then
	require.NoError(t, err)
	require.Equal(t, privateNS("u1"), cacheNS)
	require.Len(t, repMock.DeletePrivatesCalls(), 1)
}

func TestCachedPrivateRep_FetchPrivate(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchPrivateFunc = func(ctx context.Context, namespace string, username string) (stravaganza.Element, error) {
		prv := stravaganza.NewBuilder("prv0").Build()
		return prv, nil
	}

	// when
	rep := cachedPrivateRep{
		c:   cacheMock,
		rep: repMock,
	}
	prv, err := rep.FetchPrivate(context.Background(), "n0", "u1")

	// then
	require.NotNil(t, prv)
	require.NoError(t, err)

	require.Equal(t, "prv0", prv.Name())

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchPrivateCalls(), 1)
}
