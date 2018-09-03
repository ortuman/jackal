/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package rostermodel

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestModelRosterNotification(t *testing.T) {
	var rn1, rn2 Notification

	j1, _ := jid.NewWithString("ortuman@jackal.im", true)
	j2, _ := jid.NewWithString("noelia@jackal.im", true)

	rn1 = Notification{
		Contact:  "noelia",
		JID:      "ortuman@jackal.im",
		Presence: xmpp.NewPresence(j1, j2, xmpp.AvailableType),
	}
	buf := new(bytes.Buffer)
	rn1.ToGob(gob.NewEncoder(buf))
	rn2.FromGob(gob.NewDecoder(buf))
	require.Equal(t, "ortuman@jackal.im", rn2.JID)
	require.Equal(t, "noelia", rn2.Contact)
	require.NotNil(t, rn1.Presence)
	require.NotNil(t, rn2.Presence)
	require.Equal(t, rn1.Presence.String(), rn2.Presence.String())
}
