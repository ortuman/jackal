/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"bytes"
	"testing"
	"time"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestModelUser(t *testing.T) {
	var usr1 User

	j1, _ := jid.NewWithString("ortuman@jackal.im", true)
	j2, _ := jid.NewWithString("ortuman@jackal.im", true)

	usr1.Username = "ortuman"
	usr1.Password = "1234"
	usr1.LastPresence = xmpp.NewPresence(j1, j2, xmpp.AvailableType)

	buf := new(bytes.Buffer)
	require.Nil(t, usr1.ToBytes(buf))
	usr2 := User{}
	require.Nil(t, usr2.FromBytes(buf))
	require.Equal(t, usr1.Username, usr2.Username)
	require.Equal(t, usr1.Password, usr2.Password)
	require.Equal(t, usr1.LastPresence.String(), usr2.LastPresence.String())
	require.NotEqual(t, time.Time{}, usr2.LastPresenceAt)
}
