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

	capsmodel "github.com/ortuman/jackal/pkg/model/caps"
	"github.com/stretchr/testify/require"
)

func TestCachedCapsRep_UpsertCaps(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertCapabilitiesFunc = func(ctx context.Context, caps *capsmodel.Capabilities) error {
		return nil
	}

	// when
	rep := cachedCapsRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.UpsertCapabilities(context.Background(), &capsmodel.Capabilities{
		Node:     "n1",
		Ver:      "v1",
		Features: []string{"f0"},
	})

	// then
	require.NoError(t, err)
	require.Equal(t, capsNS("n1", "v1"), cacheNS)
	require.Equal(t, capsKey, cacheKey)
	require.Len(t, repMock.UpsertCapabilitiesCalls(), 1)
}

func TestCachedCapsRep_CapsExist(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.HasKeyFunc = func(ctx context.Context, ns, k string) (bool, error) {
		if ns == capsNS("n1", "v1") && k == capsKey {
			return true, nil
		}
		return false, nil
	}

	repMock := &repositoryMock{}
	repMock.CapabilitiesExistFunc = func(ctx context.Context, node, ver string) (bool, error) {
		return node == "n2" && ver == "v2", nil
	}

	// when
	rep := cachedCapsRep{
		c:   cacheMock,
		rep: repMock,
	}
	ok1, err1 := rep.CapabilitiesExist(context.Background(), "n1", "v1")
	ok2, err2 := rep.CapabilitiesExist(context.Background(), "n2", "v2")
	ok3, err3 := rep.CapabilitiesExist(context.Background(), "n3", "v3")

	// then
	require.True(t, ok1)
	require.NoError(t, err1)

	require.True(t, ok2)
	require.NoError(t, err2)

	require.False(t, ok3)
	require.NoError(t, err3)

	require.Len(t, repMock.CapabilitiesExistCalls(), 2)
}

func TestCachedCapsRep_FetchCaps(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchCapabilitiesFunc = func(ctx context.Context, node, ver string) (*capsmodel.Capabilities, error) {
		return &capsmodel.Capabilities{
			Node: "n1",
			Ver:  "v1",
		}, nil
	}

	// when
	rep := cachedCapsRep{
		c:   cacheMock,
		rep: repMock,
	}
	caps, err := rep.FetchCapabilities(context.Background(), "n1", "v1")

	// then
	require.NotNil(t, caps)
	require.NoError(t, err)

	require.Equal(t, "n1", caps.Node)
	require.Equal(t, "v1", caps.Ver)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchCapabilitiesCalls(), 1)
}
