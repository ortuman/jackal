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

	lastmodel "github.com/ortuman/jackal/pkg/model/last"

	"github.com/stretchr/testify/require"
)

func TestCachedLastRep_UpsertLast(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertLastFunc = func(ctx context.Context, last *lastmodel.Last) error {
		return nil
	}

	// when
	rep := cachedLastRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.UpsertLast(context.Background(), &lastmodel.Last{Username: "u1"})

	// then
	require.NoError(t, err)
	require.Equal(t, lastNS("u1"), cacheNS)
	require.Equal(t, lastKey, cacheKey)
	require.Len(t, repMock.UpsertLastCalls(), 1)
}

func TestCachedLastRep_DeleteLast(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteLastFunc = func(ctx context.Context, username string) error {
		return nil
	}

	// when
	rep := cachedLastRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteLast(context.Background(), "v1")

	// then
	require.NoError(t, err)
	require.Equal(t, lastNS("v1"), cacheNS)
	require.Equal(t, lastKey, cacheKey)
	require.Len(t, repMock.DeleteLastCalls(), 1)
}

func TestCachedLastRep_FetchLast(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchLastFunc = func(ctx context.Context, username string) (*lastmodel.Last, error) {
		return &lastmodel.Last{Username: "u1"}, nil
	}

	// when
	rep := cachedLastRep{
		c:   cacheMock,
		rep: repMock,
	}
	last, err := rep.FetchLast(context.Background(), "u1")

	// then
	require.NotNil(t, last)
	require.NoError(t, err)

	require.Equal(t, "u1", last.Username)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchLastCalls(), 1)
}
