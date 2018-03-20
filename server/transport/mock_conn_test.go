/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bytes"
	"testing"
	"time"

	"github.com/ortuman/jackal/util"
	"github.com/stretchr/testify/require"
)

func TestMockConn(t *testing.T) {
	mc := NewMockConn()

	bt := util.RandomBytes(256)
	mc.Write(bt)
	require.Equal(t, 0, bytes.Compare(bt, mc.ReadBytes()))

	bt = util.RandomBytes(256)
	mc.SendBytes(bt)
	bt2 := make([]byte, 256)
	mc.Read(bt2)
	require.Equal(t, 0, bytes.Compare(bt, bt2))

	require.Equal(t, mockConnNetwork, mc.LocalAddr().Network())
	require.Equal(t, mockConnLocalAddr, mc.LocalAddr().String())
	require.Equal(t, mockConnNetwork, mc.RemoteAddr().Network())
	require.Equal(t, mockConnRemoteAddr, mc.RemoteAddr().String())

	mc.Close()
	require.True(t, mc.IsClosed())

	require.Nil(t, mc.SetDeadline(time.Now()))
	require.Nil(t, mc.SetReadDeadline(time.Now()))
	require.Nil(t, mc.SetWriteDeadline(time.Now()))
}
