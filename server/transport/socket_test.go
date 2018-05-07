/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bytes"
	"crypto/tls"
	"testing"

	"github.com/ortuman/jackal/server/compress"
	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

func TestSocket(t *testing.T) {
	mc := NewMockConn()
	st := NewSocketTransport(mc, 4096, 120)
	st2 := st.(*socketTransport)

	el1 := xml.NewElementNamespace("elem", "exodus:ns")
	st.WriteElement(el1, true)
	require.Equal(t, 0, bytes.Compare([]byte(el1.String()), mc.ClientReadBytes()))

	el2 := xml.NewElementNamespace("elem2", "exodus2:ns")
	mc.ClientWriteBytes([]byte(el2.String()))
	el3, err := st.ReadElement()
	require.Nil(t, err)
	require.NotNil(t, el3)
	require.Equal(t, el2.String(), el3.String())

	st.EnableCompression(compress.BestCompression)
	require.True(t, st2.compressionEnabled)

	st.StartTLS(&tls.Config{})
	_, ok := st2.conn.(*tls.Conn)
	require.True(t, ok)

	require.Nil(t, st2.ChannelBindingBytes(ChannelBindingMechanism(99)))
	require.Nil(t, st2.ChannelBindingBytes(TLSUnique))

	st.Close()
	require.True(t, mc.IsClosed())
}
