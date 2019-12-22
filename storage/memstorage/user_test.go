/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertUser(t *testing.T) {
	u := model.User{Username: "ortuman", Password: "1234"}
	s := New()
	s.EnableMockedError()
	err := s.UpsertUser(&u)
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	err = s.UpsertUser(&u)
	require.Nil(t, err)
}

func TestMemoryStorage_UserExists(t *testing.T) {
	s := New()
	s.EnableMockedError()
	_, err := s.UserExists("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	ok, err := s.UserExists("ortuman")
	require.Nil(t, err)
	require.False(t, ok)
}

func TestMemoryStorage_FetchUser(t *testing.T) {
	u := model.User{Username: "ortuman", Password: "1234"}
	s := New()
	_ = s.UpsertUser(&u)

	s.EnableMockedError()
	_, err := s.FetchUser("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()

	usr, _ := s.FetchUser("romeo")
	require.Nil(t, usr)

	usr, _ = s.FetchUser("ortuman")
	require.NotNil(t, usr)
}

func TestMemoryStorage_DeleteUser(t *testing.T) {
	u := model.User{Username: "ortuman", Password: "1234"}
	s := New()
	_ = s.UpsertUser(&u)

	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.DeleteUser("ortuman"))
	s.DisableMockedError()
	require.Nil(t, s.DeleteUser("ortuman"))

	usr, _ := s.FetchUser("ortuman")
	require.Nil(t, usr)
}
