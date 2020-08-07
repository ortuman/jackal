/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelRoomConfig(t *testing.T){
	rc1 := RoomConfig{
		Public: true,
		Persistent: true,
		PwdProtected: true,
		Password: "pwd",
		Open: true,
		Moderated: true,
		NonAnonymous: false,
	}

	buf := new(bytes.Buffer)
	require.Nil(t, rc1.ToBytes(buf))

	rc2 := RoomConfig{}
	require.Nil(t, rc2.FromBytes(buf))
	require.Equal(t, rc1.Public, rc2.Public)
	require.Equal(t, rc1.Persistent, rc2.Persistent)
	require.Equal(t, rc1.PwdProtected, rc2.PwdProtected)
	require.Equal(t, rc1.Password, rc2.Password)
	require.Equal(t, rc1.Open, rc2.Open)
	require.Equal(t, rc1.Moderated, rc2.Moderated)
	require.Equal(t, rc1.NonAnonymous, rc2.NonAnonymous)
}
