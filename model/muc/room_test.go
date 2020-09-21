/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"testing"

	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelRoom(t *testing.T) {
	rJID, _ := jid.NewWithString("room@conference.jackal.im", true)
	rc := RoomConfig{
		Open: true,
	}
	jFull, _ := jid.NewWithString("ortuman@jackal.im/laptop", true)
	o := &Occupant{
		BareJID:     jFull,
		OccupantJID: jFull,
	}
	userMap := make(map[jid.JID]jid.JID)
	userMap[*o.BareJID.ToBareJID()] = *jFull

	invitedMap := make(map[jid.JID]bool)
	invitedMap[*o.BareJID.ToBareJID()] = true

	r1 := Room{
		Config:         &rc,
		Name:           "Test Room",
		RoomJID:        rJID,
		Desc:           "Test Description",
		Subject:        "Test Subject",
		Language:       "eng",
		UserToOccupant: userMap,
		InvitedUsers:   invitedMap,
		Locked:         true,
	}

	buf := new(bytes.Buffer)
	require.Nil(t, r1.ToBytes(buf))

	r2 := Room{}
	require.Nil(t, r2.FromBytes(buf))
	assert.EqualValues(t, r1, r2)

	newJID, _ := jid.NewWithString("milos@jackal.im/laptop", true)
	o2 := &Occupant{
		BareJID:     newJID,
		OccupantJID: newJID,
	}
	require.Equal(t, len(r2.UserToOccupant), 1)
	r2.AddOccupant(o2)
	require.Equal(t, len(r2.UserToOccupant), 2)
}
