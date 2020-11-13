/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
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

func TestRoom_Bytes(t *testing.T) {
	r1 := getTestRoom()
	r1.userToOccupant = make(map[jid.JID]jid.JID)
	r1.invitedUsers = make(map[jid.JID]bool)
	buf := new(bytes.Buffer)
	require.Nil(t, r1.ToBytes(buf))

	r2 := &Room{}
	require.Nil(t, r2.FromBytes(buf))
	assert.EqualValues(t, r1, r2)
}

func TestRoom_Occupants(t *testing.T) {
	room := getTestRoom()
	userJID, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)
	occJID, _ := jid.NewWithString("testroom@conference.jackal.im/nick", true)
	o := &Occupant{
		OccupantJID: occJID,
		BareJID:     userJID.ToBareJID(),
		affiliation: "member",
	}

	room.AddOccupant(o)
	require.True(t, o.IsParticipant())
	require.True(t, room.UserIsInRoom(userJID.ToBareJID()))
	require.Equal(t, room.occupantsOnline, 1)
	resJID, inRoom := room.GetOccupantJID(userJID.ToBareJID())
	require.Equal(t, resJID.String(), occJID.String())
	require.True(t, inRoom)
	room.OccupantLeft(o)
	require.True(t, room.UserIsInRoom(userJID.ToBareJID()))
	require.Equal(t, room.occupantsOnline, 0)
}

func TestRoom_Invites(t *testing.T) {
	room := getTestRoom()
	userJID, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)

	require.False(t, room.UserIsInvited(userJID.ToBareJID()))
	err := room.InviteUser(userJID.ToBareJID())
	require.Nil(t, err)
	require.True(t, room.UserIsInvited(userJID.ToBareJID()))
	room.DeleteInvite(userJID.ToBareJID())
	require.False(t, room.UserIsInvited(userJID.ToBareJID()))
}

func getTestRoom() *Room {
	rc := RoomConfig{
		Public:       true,
		Persistent:   true,
		PwdProtected: false,
		Open:         true,
		Moderated:    true,
	}
	j, _ := jid.NewWithString("testroom@conference.jackal.im", true)
	return &Room{
		Name:    "testRoom",
		RoomJID: j,
		Desc:    "Room for Testing",
		Config:  &rc,
		Locked:  false,
	}
}
