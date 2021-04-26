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

package measuredrepository

import (
	"context"
	"testing"

	coremodel "github.com/ortuman/jackal/pkg/model/core"
	"github.com/stretchr/testify/require"
)

func TestMeasuredUserRep_UpsertUser(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UpsertUserFunc = func(ctx context.Context, user *coremodel.User) error {
		return nil
	}
	m := New(repMock)

	// when
	_ = m.UpsertUser(context.Background(), &coremodel.User{})

	// then
	require.Len(t, repMock.UpsertUserCalls(), 1)
}

func TestMeasuredUserRep_DeleteUser(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.DeleteUserFunc = func(ctx context.Context, username string) error {
		return nil
	}
	m := New(repMock)

	// when
	_ = m.DeleteUser(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.DeleteUserCalls(), 1)
}

func TestMeasuredUserRep_FetchUser(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.FetchUserFunc = func(ctx context.Context, username string) (*coremodel.User, error) {
		return &coremodel.User{}, nil
	}
	m := New(repMock)

	// when
	_, _ = m.FetchUser(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.FetchUserCalls(), 1)

}

func TestMeasuredUserRep_UserExists(t *testing.T) {
	// given
	repMock := &repositoryMock{}
	repMock.UserExistsFunc = func(ctx context.Context, username string) (bool, error) {
		return true, nil
	}
	m := New(repMock)

	// when
	_, _ = m.UserExists(context.Background(), "ortuman")

	// then
	require.Len(t, repMock.UserExistsCalls(), 1)
}
