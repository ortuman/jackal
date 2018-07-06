/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_User(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	usr := model.User{Username: "ortuman", Password: "1234"}

	err := h.db.InsertOrUpdateUser(&usr)
	require.Nil(t, err)

	usr2, err := h.db.FetchUser("ortuman")
	require.Nil(t, err)
	require.Equal(t, "ortuman", usr2.Username)
	require.Equal(t, "1234", usr2.Password)

	exists, err := h.db.UserExists("ortuman")
	require.Nil(t, err)
	require.True(t, exists)

	usr3, err := h.db.FetchUser("ortuman2")
	require.Nil(t, usr3)
	require.Nil(t, err)

	err = h.db.DeleteUser("ortuman")
	require.Nil(t, err)

	exists, err = h.db.UserExists("ortuman")
	require.Nil(t, err)
	require.False(t, exists)
}
