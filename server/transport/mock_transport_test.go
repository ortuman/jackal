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
	"github.com/ortuman/jackal/util"
	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

func TestMockTransport(t *testing.T) {
	tr := NewMockTransport()

	el1 := xml.NewElementNamespace("elem", "exodus:ns")
	tr.WriteElement(el1, true)
	require.Equal(t, 0, bytes.Compare([]byte(el1.String()), tr.GetWrittenBytes()))

	el2 := xml.NewElementNamespace("elem2", "exodus2:ns")
	tr.SetReadBytes([]byte(el2.String()))
	el3, err := tr.ReadElement()
	require.Nil(t, err)
	require.NotNil(t, el3)
	require.Equal(t, el2.String(), el3.String())

	bt := util.RandomBytes(256)
	tr.SetChannelBindingBytes(bt)
	require.Equal(t, 0, bytes.Compare(tr.ChannelBindingBytes(TLSUnique), bt))

	tr.StartTLS(&tls.Config{})
	require.True(t, tr.IsSecured())

	tr.EnableCompression(compress.BestCompression)
	require.True(t, tr.IsCompressed())

	tr.Close()
	require.True(t, tr.IsClosed())
}
