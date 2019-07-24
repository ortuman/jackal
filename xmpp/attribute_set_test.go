/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttributeSet_SetAndGet(t *testing.T) {
	as := attributeSet{}
	as.setAttribute("id", "1234")
	require.Equal(t, "1234", as.Get("id"))
	require.Equal(t, "", as.Get("id2"))
}

func TestAttributeSet_Remove(t *testing.T) {
	as := attributeSet{}
	as.setAttribute("id", "1234")
	require.Equal(t, "1234", as.Get("id"))
	as.removeAttribute("id")
	require.Equal(t, "", as.Get("id"))
}

func TestAttributeSet_Gob(t *testing.T) {
	as := attributeSet{}
	as.setAttribute("a", "1234")
	as.setAttribute("b", "5678")
	buf := new(bytes.Buffer)
	require.Nil(t, as.ToBytes(buf))

	expected := []byte{3, 4, 0, 4, 4, 12, 0, 1, 97, 7, 12, 0, 4, 49, 50, 51, 52, 4, 12, 0, 1, 98, 7, 12, 0, 4, 53, 54, 55, 56}
	require.Equal(t, 0, bytes.Compare(expected, buf.Bytes()))

	as2 := attributeSet{}
	require.Nil(t, as2.FromBytes(buf))
	require.Equal(t, as, as2)
}
