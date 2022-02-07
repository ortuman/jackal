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

	"github.com/stretchr/testify/require"
)

func TestCachedRosterRep_TouchVersion(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.TouchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 5, nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	ver, err := rep.TouchRosterVersion(context.Background(), "u1")

	// then
	require.NoError(t, err)
	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Equal(t, rosterVersionKey, cacheKey)
	require.Equal(t, 5, ver)
	require.Len(t, repMock.TouchRosterVersionCalls(), 1)
}

func TestCachedUserRep_FetchRosterVersion(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		cacheNS = ns
		cacheKey = k
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchRosterVersionFunc = func(ctx context.Context, username string) (int, error) {
		return 5, nil
	}

	// when
	rep := cachedRosterRep{
		c:   cacheMock,
		rep: repMock,
	}
	ver, err := rep.FetchRosterVersion(context.Background(), "u1")

	// then
	require.NoError(t, err)
	require.Equal(t, 5, ver)

	require.Equal(t, rosterItemsNS("u1"), cacheNS)
	require.Equal(t, rosterVersionKey, cacheKey)
	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchRosterVersionCalls(), 1)
}
