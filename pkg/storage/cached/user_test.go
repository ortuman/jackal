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

	usermodel "github.com/ortuman/jackal/pkg/model/user"
	"github.com/stretchr/testify/require"
)

func TestCachedUserRep_UpsertUser(t *testing.T) {
	// given
	var cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, k string) error {
		cacheKey = k
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
	require.Equal(t, userKey("u1"), cacheKey)
	require.Len(t, repMock.UpsertUserCalls(), 1)
}

func TestCachedUserRep_DeleteUser(t *testing.T) {
	// given
	var cacheKey string

	cacheMock := &cacheMock{}
	cacheMock.DelFunc = func(ctx context.Context, k string) error {
		cacheKey = k
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
	require.Equal(t, userKey("u1"), cacheKey)
	require.Len(t, repMock.DeleteUserCalls(), 1)
}

func TestCachedUserRep_FetchUser(t *testing.T) {

}

func TestCachedUserRep_UserExists(t *testing.T) {

}
