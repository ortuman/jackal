/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"testing"

	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXEP0045_CreateRoom(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	occJID, _ := jid.New("room", "conference.jackal.im", "nick", true)
	fullJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	o, err := muc.createOwner(nil, fullJID, occJID)
	require.Nil(t, err)

	roomJID, _ := jid.New("room", "conference.jackal.im", "", true)
	room, err := muc.createRoom(nil, roomJID, o)
	require.Nil(t, err)
	require.NotNil(t, room)
	require.True(t, room.UserIsInRoom(fullJID.ToBareJID()))
	jidInRoom, _ := room.GetOccupantJID(fullJID.ToBareJID())
	assert.EqualValues(t, jidInRoom, *occJID)

	roomMem, err := c.Room().FetchRoom(nil, roomJID)
	require.Nil(t, err)
	require.Equal(t, roomJID.String(), roomMem.RoomJID.String())
}

func TestXEP0045_NewRoom(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)
	err := muc.newRoom(nil, from, to)
	require.Nil(t, err)

	roomMem, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.NotNil(t, roomMem)
	assert.EqualValues(t, to.ToBareJID(), roomMem.RoomJID)
	toRoom, _ := roomMem.GetOccupantJID(from.ToBareJID())
	assert.EqualValues(t, *to, toRoom)
	require.Equal(t, muc.allRooms[0].String(), to.ToBareJID().String())

	oMem, err := c.Occupant().FetchOccupant(nil, to)
	require.Nil(t, err)
	require.NotNil(t, oMem)
	assert.EqualValues(t, from.ToBareJID(), oMem.BareJID)
}
