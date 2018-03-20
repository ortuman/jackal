/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bytes"
	"crypto/tls"
	"testing"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/util"
	"github.com/stretchr/testify/require"
)

func TestSocket(t *testing.T) {
	mc := NewMockConn()
	st := NewSocketTransport(mc, 4096, 120)
	st2 := st.(*socketTransport)

	bt := util.RandomBytes(256)
	st.Write(bt)
	require.Equal(t, 0, bytes.Compare(bt, mc.ReadBytes()))

	bt = util.RandomBytes(256)
	mc.SendBytes(bt)
	b2 := make([]byte, 256)

	st.Read(b2)
	require.Equal(t, 0, bytes.Compare(bt, b2))

	st.EnableCompression(config.BestCompression)
	require.True(t, st2.compressionEnabled)

	st.StartTLS(&tls.Config{})
	_, ok := st2.conn.(*tls.Conn)
	require.True(t, ok)

	require.Nil(t, st2.ChannelBindingBytes(config.ChannelBindingMechanism(99)))
	require.Nil(t, st2.ChannelBindingBytes(config.TLSUnique))

	st.Close()
	require.True(t, mc.IsClosed())
}
