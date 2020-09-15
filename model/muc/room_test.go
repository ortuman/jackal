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
		Nick:        "mynick",
		BareJID:     jFull,
		OccupantJID: jFull,
	}
	userMap := make(map[jid.JID]jid.JID)
	userMap[*o.BareJID.ToBareJID()] = *jFull

	r1 := Room{
		Config:            &rc,
		Name:              "Test Room",
		RoomJID:           rJID,
		Desc:              "Test Description",
		Subject:           "Test Subject",
		Language:          "eng",
		numberOfOccupants: 1,
		UserToOccupant:    userMap,
		Locked:            true,
	}

	buf := new(bytes.Buffer)
	require.Nil(t, r1.ToBytes(buf))

	r2 := Room{}
	require.Nil(t, r2.FromBytes(buf))
	requireRoomsAreEqual(t, r1, r2)

	newJID, _ := jid.NewWithString("milos@jackal.im/laptop", true)
	o2 := &Occupant{
		Nick:        "milos",
		BareJID:     newJID,
		OccupantJID: newJID,
	}
	require.Equal(t, r2.numberOfOccupants, 1)
	r2.AddOccupant(o2)
	require.Equal(t, r2.numberOfOccupants, 2)
}

func requireRoomsAreEqual(t *testing.T, r1, r2 Room) {
	assert.EqualValues(t, *r1.Config, *r2.Config)
	require.Equal(t, r1.Name, r2.Name)
	require.Equal(t, r1.Desc, r2.Desc)
	require.Equal(t, r1.Subject, r2.Subject)
	require.Equal(t, r1.Language, r2.Language)
	require.Equal(t, r1.Locked, r2.Locked)
	require.Equal(t, r1.numberOfOccupants, r2.numberOfOccupants)
	require.Equal(t, r1.RoomJID.String(), r2.RoomJID.String())
}
