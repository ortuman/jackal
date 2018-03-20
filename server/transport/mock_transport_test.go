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

func TestMockTransport(t *testing.T) {
	tr := NewMockTransport()

	bt := util.RandomBytes(256)
	tr.Write(bt)
	require.Equal(t, 0, bytes.Compare(bt, tr.GetWrittenBytes()))

	bt = util.RandomBytes(256)
	tr.SetReadBytes(bt)
	bt2 := make([]byte, 256)
	tr.Read(bt2)
	require.Equal(t, 0, bytes.Compare(bt, bt2))

	bt3 := util.RandomBytes(256)
	tr.SetChannelBindingBytes(bt3)
	require.Equal(t, 0, bytes.Compare(tr.ChannelBindingBytes(config.TLSUnique), bt3))

	tr.StartTLS(&tls.Config{})
	require.True(t, tr.IsSecured())

	tr.EnableCompression(config.BestCompression)
	require.True(t, tr.IsCompressed())

	tr.Close()
	require.True(t, tr.IsClosed())
}
