/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_User(t *testing.T) {
	t.Parallel()

	s, teardown := newUserMock()
	defer teardown()

	usr := model.User{Username: "ortuman", Password: "1234"}

	err := s.UpsertUser(context.Background(), &usr)
	require.Nil(t, err)

	usr2, err := s.FetchUser(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, "ortuman", usr2.Username)
	require.Equal(t, "1234", usr2.Password)

	exists, err := s.UserExists(context.Background(), "ortuman")
	require.Nil(t, err)
	require.True(t, exists)

	usr3, err := s.FetchUser(context.Background(), "ortuman2")
	require.Nil(t, usr3)
	require.Nil(t, err)

	err = s.DeleteUser(context.Background(), "ortuman")
	require.Nil(t, err)

	exists, err = s.UserExists(context.Background(), "ortuman")
	require.Nil(t, err)
	require.False(t, exists)
}

func newUserMock() (*badgerDBUser, func()) {
	t := newT()
	return &badgerDBUser{badgerDBStorage: newStorage(t.DB)}, func() {
		t.teardown()
	}
}
