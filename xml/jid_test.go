/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml_test

import (
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

func TestBadJID(t *testing.T) {
	_, err := xml.NewJIDString("ortuman@", false)
	require.NotNil(t, err)
	longStr := ""
	for i := 0; i < 1074; i++ {
		longStr += "a"
	}
	_, err2 := xml.NewJID(longStr, "example.org", "res", false)
	require.NotNil(t, err2)
	_, err3 := xml.NewJID("ortuman", longStr, "res", false)
	require.NotNil(t, err3)
	_, err4 := xml.NewJID("ortuman", "example.org", longStr, false)
	require.NotNil(t, err4)
}

func TestNewJID(t *testing.T) {
	j1, err := xml.NewJID("ortuman", "example.org", "res", false)
	require.Nil(t, err)
	require.Equal(t, "ortuman", j1.Node())
	require.Equal(t, "example.org", j1.Domain())
	require.Equal(t, "res", j1.Resource())
	j2, err := xml.NewJID("ortuman", "example.org", "res", true)
	require.Nil(t, err)
	require.Equal(t, "ortuman", j2.Node())
	require.Equal(t, "example.org", j2.Domain())
	require.Equal(t, "res", j2.Resource())
}

func TestEmptyJID(t *testing.T) {
	j, err := xml.NewJIDString("", true)
	require.Nil(t, err)
	require.Equal(t, "", j.Node())
	require.Equal(t, "", j.Domain())
	require.Equal(t, "", j.Resource())
}

func TestNewJIDString(t *testing.T) {
	j, err := xml.NewJIDString("ortuman@jackal.im/res", false)
	require.Nil(t, err)
	require.Equal(t, "ortuman", j.Node())
	require.Equal(t, "jackal.im", j.Domain())
	require.Equal(t, "res", j.Resource())
	require.Equal(t, "ortuman@jackal.im", j.ToBareJID().String())
	require.Equal(t, "ortuman@jackal.im/res", j.String())
}

func TestJIDEqual(t *testing.T) {
	j1, _ := xml.NewJIDString("ortuman@jackal.im/res", false)
	j2, _ := xml.NewJID("ortuman", "jackal.im", "res", false)
	require.NotNil(t, j1)
	require.NotNil(t, j2)
	require.True(t, j1.IsEqual(j2))
}

func TestServerJID(t *testing.T) {
	j1, _ := xml.NewJIDString("example.org", false)
	j2, _ := xml.NewJIDString("user@example.org", false)
	j3, _ := xml.NewJIDString("example.org/res", false)
	require.True(t, j1.IsServer())
	require.False(t, j2.IsServer())
	require.True(t, j3.IsServer())
}

func TestBareJID(t *testing.T) {
	j1, _ := xml.NewJID("ortuman", "example.org", "res", false)
	require.True(t, j1.ToBareJID().IsBare())
	j2, _ := xml.NewJIDString("example.org/res", false)
	require.False(t, j2.ToBareJID().IsBare())
}

func TestFullJID(t *testing.T) {
	j1, _ := xml.NewJID("ortuman", "example.org", "res", false)
	require.True(t, j1.IsFull())
}

func TestEqualJID(t *testing.T) {
	j1, _ := xml.NewJIDString("ortuman@example.org/res1", false)
	j2, _ := xml.NewJIDString("user@example.org", false)
	j3, _ := xml.NewJIDString("ortuman@test.org", false)
	j4, _ := xml.NewJIDString("ortuman@example.org/res2", false)
	j5, _ := xml.NewJIDString("ortuman@example.org/res1", false)
	require.False(t, j1.IsEqual(j2))
	require.False(t, j1.IsEqual(j3))
	require.False(t, j1.IsEqual(j4))
	require.True(t, j1.IsEqual(j1))
	require.True(t, j1.IsEqual(j5))
}

func TestBadPrep(t *testing.T) {
	badNode := string([]byte{255, 255, 255})
	badDomain := "\U0001f480"
	basResource := string([]byte{255, 255, 255})
	j, err := xml.NewJID(badNode, "example.org", "res", false)
	require.Nil(t, j)
	require.NotNil(t, err)
	j2, err := xml.NewJID("ortuman", badDomain, "res", false)
	require.Nil(t, j2)
	require.NotNil(t, err)
	j3, err := xml.NewJID("ortuman", "example.org", basResource, false)
	require.Nil(t, j3)
	require.NotNil(t, err)
}
