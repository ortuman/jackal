/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package jid_test

import (
	"testing"

	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestBadJID(t *testing.T) {
	_, err := jid.NewWithString("ortuman@", false)
	require.NotNil(t, err)
	longStr := ""
	for i := 0; i < 1074; i++ {
		longStr += "a"
	}
	_, err2 := jid.New(longStr, "example.org", "res", false)
	require.NotNil(t, err2)
	_, err3 := jid.New("ortuman", longStr, "res", false)
	require.NotNil(t, err3)
	_, err4 := jid.New("ortuman", "example.org", longStr, false)
	require.NotNil(t, err4)
}

func TestNewJID(t *testing.T) {
	j1, err := jid.New("ortuman", "example.org", "res", false)
	require.Nil(t, err)
	require.Equal(t, "ortuman", j1.Node())
	require.Equal(t, "example.org", j1.Domain())
	require.Equal(t, "res", j1.Resource())
	j2, err := jid.New("ortuman", "example.org", "res", true)
	require.Nil(t, err)
	require.Equal(t, "ortuman", j2.Node())
	require.Equal(t, "example.org", j2.Domain())
	require.Equal(t, "res", j2.Resource())
}

func TestEmptyJID(t *testing.T) {
	j, err := jid.NewWithString("", true)
	require.Nil(t, err)
	require.Equal(t, "", j.Node())
	require.Equal(t, "", j.Domain())
	require.Equal(t, "", j.Resource())
}

func TestNewJIDString(t *testing.T) {
	j, err := jid.NewWithString("ortuman@jackal.im/res", false)
	require.Nil(t, err)
	require.Equal(t, "ortuman", j.Node())
	require.Equal(t, "jackal.im", j.Domain())
	require.Equal(t, "res", j.Resource())
	require.Equal(t, "ortuman@jackal.im", j.ToBareJID().String())
	require.Equal(t, "ortuman@jackal.im/res", j.String())
}

func TestServerJID(t *testing.T) {
	j1, _ := jid.NewWithString("example.org", false)
	j2, _ := jid.NewWithString("user@example.org", false)
	j3, _ := jid.NewWithString("example.org/res", false)
	require.True(t, j1.IsServer())
	require.False(t, j2.IsServer())
	require.True(t, j3.IsServer() && j3.IsFull())
}

func TestBareJID(t *testing.T) {
	j1, _ := jid.New("ortuman", "example.org", "res", false)
	require.True(t, j1.ToBareJID().IsBare())
	j2, _ := jid.NewWithString("example.org/res", false)
	require.False(t, j2.ToBareJID().IsBare())
}

func TestFullJID(t *testing.T) {
	j1, _ := jid.New("ortuman", "example.org", "res", false)
	j2, _ := jid.New("", "example.org", "res", false)
	require.True(t, j1.IsFullWithUser())
	require.True(t, j2.IsFullWithServer())
}

func TestMatchesJID(t *testing.T) {
	j1, _ := jid.NewWithString("ortuman@example.org/res1", false)
	j2, _ := jid.NewWithString("ortuman@example.org", false)
	j3, _ := jid.NewWithString("example.org", false)
	j4, _ := jid.NewWithString("example.org/res1", false)
	j6, _ := jid.NewWithString("ortuman@example2.org/res2", false)
	require.True(t, j1.Matches(j1, jid.MatchesNode|jid.MatchesDomain|jid.MatchesResource))
	require.True(t, j1.Matches(j2, jid.MatchesNode|jid.MatchesDomain))
	require.True(t, j1.Matches(j3, jid.MatchesDomain))
	require.True(t, j1.Matches(j4, jid.MatchesDomain|jid.MatchesResource))

	require.False(t, j1.Matches(j2, jid.MatchesNode|jid.MatchesDomain|jid.MatchesResource))
	require.False(t, j6.Matches(j2, jid.MatchesNode|jid.MatchesDomain))
	require.False(t, j6.Matches(j3, jid.MatchesDomain))
	require.False(t, j6.Matches(j4, jid.MatchesDomain|jid.MatchesResource))
}

func TestBadPrep(t *testing.T) {
	badNode := string([]byte{255, 255, 255})
	badDomain := string([]byte{255, 255, 255})
	badResource := string([]byte{255, 255, 255})
	j, err := jid.New(badNode, "example.org", "res", false)
	require.Nil(t, j)
	require.NotNil(t, err)
	j2, err := jid.New("ortuman", badDomain, "res", false)
	require.Nil(t, j2)
	require.NotNil(t, err)
	j3, err := jid.New("ortuman", "example.org", badResource, false)
	require.Nil(t, j3)
	require.NotNil(t, err)
}
