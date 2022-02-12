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

	usermodel "github.com/ortuman/jackal/pkg/model/user"
	"github.com/stretchr/testify/require"
)

func TestCachedUserRep_UpsertUser(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.UpsertUserFunc = func(ctx context.Context, user *usermodel.User) error {
		return nil
	}

	// when
	rep := cachedUserRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.UpsertUser(context.Background(), &usermodel.User{Username: "u1"})

	// then
	require.NoError(t, err)
	require.Equal(t, userNS("u1"), cacheNS)
	require.Equal(t, userKey, cacheKey)
	require.Len(t, repMock.UpsertUserCalls(), 1)
}

func TestCachedUserRep_DeleteUser(t *testing.T) {
	// given
	var cacheNS, cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, ns string, keys ...string) error {
		cacheNS = ns
		cacheKey = keys[0]
		return nil
	}

	repMock := &repositoryMock{}
	repMock.DeleteUserFunc = func(ctx context.Context, username string) error {
		return nil
	}

	// when
	rep := cachedUserRep{
		c:   cacheMock,
		rep: repMock,
	}
	err := rep.DeleteUser(context.Background(), "u1")

	// then
	require.NoError(t, err)
	require.Equal(t, userNS("u1"), cacheNS)
	require.Equal(t, userKey, cacheKey)
	require.Len(t, repMock.DeleteUserCalls(), 1)
}

func TestCachedUserRep_FetchUser(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.GetFunc = func(ctx context.Context, ns, k string) ([]byte, error) {
		return nil, nil
	}
	cacheMock.PutFunc = func(ctx context.Context, ns, k string, val []byte) error {
		return nil
	}

	repMock := &repositoryMock{}
	repMock.FetchUserFunc = func(ctx context.Context, username string) (*usermodel.User, error) {
		return &usermodel.User{Username: "u1"}, nil
	}

	// when
	rep := cachedUserRep{
		c:   cacheMock,
		rep: repMock,
	}
	usr, err := rep.FetchUser(context.Background(), "u1")

	// then
	require.NotNil(t, usr)
	require.NoError(t, err)

	require.Equal(t, "u1", usr.Username)

	require.Len(t, cacheMock.GetCalls(), 1)
	require.Len(t, cacheMock.PutCalls(), 1)
	require.Len(t, repMock.FetchUserCalls(), 1)
}

func TestCachedUserRep_UserExists(t *testing.T) {
	// given
	cacheMock := &cacheMock{}
	cacheMock.HasKeyFunc = func(ctx context.Context, ns, k string) (bool, error) {
		if ns == userNS("u1") && k == userKey {
			return true, nil
		}
		return false, nil
	}

	repMock := &repositoryMock{}
	repMock.UserExistsFunc = func(ctx context.Context, username string) (bool, error) {
		return username == "u2", nil
	}

	// when
	rep := cachedUserRep{
		c:   cacheMock,
		rep: repMock,
	}
	ok1, err1 := rep.UserExists(context.Background(), "u1")
	ok2, err2 := rep.UserExists(context.Background(), "u2")
	ok3, err3 := rep.UserExists(context.Background(), "u3")

	// then
	require.True(t, ok1)
	require.NoError(t, err1)

	require.True(t, ok2)
	require.NoError(t, err2)

	require.False(t, ok3)
	require.NoError(t, err3)

	require.Len(t, repMock.UserExistsCalls(), 2)
}
