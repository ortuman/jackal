/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memory

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertUser(t *testing.T) {
	u := model.User{Username: "ortuman", Password: "1234"}
	s := newUser()
	s.EnableMockedError()
	err := s.UpsertUser(context.Background(), &u)
	require.Equal(t, errMocked, err)
	s.DisableMockedError()
	err = s.UpsertUser(context.Background(), &u)
	require.Nil(t, err)
}

func TestMemoryStorage_UserExists(t *testing.T) {
	s := newUser()
	s.EnableMockedError()
	_, err := s.UserExists(context.Background(), "ortuman")
	require.Equal(t, errMocked, err)
	s.DisableMockedError()
	ok, err := s.UserExists(context.Background(), "ortuman")
	require.Nil(t, err)
	require.False(t, ok)
}

func TestMemoryStorage_FetchUser(t *testing.T) {
	u := model.User{Username: "ortuman", Password: "1234"}
	s := newUser()
	_ = s.UpsertUser(context.Background(), &u)

	s.EnableMockedError()
	_, err := s.FetchUser(context.Background(), "ortuman")
	require.Equal(t, errMocked, err)
	s.DisableMockedError()

	usr, _ := s.FetchUser(context.Background(), "romeo")
	require.Nil(t, usr)

	usr, _ = s.FetchUser(context.Background(), "ortuman")
	require.NotNil(t, usr)
}

func TestMemoryStorage_DeleteUser(t *testing.T) {
	u := model.User{Username: "ortuman", Password: "1234"}
	s := newUser()
	_ = s.UpsertUser(context.Background(), &u)

	s.EnableMockedError()
	require.Equal(t, errMocked, s.DeleteUser(context.Background(), "ortuman"))
	s.DisableMockedError()
	require.Nil(t, s.DeleteUser(context.Background(), "ortuman"))

	usr, _ := s.FetchUser(context.Background(), "ortuman")
	require.Nil(t, usr)
}
