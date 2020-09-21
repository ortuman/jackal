/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestXEP0045_CreateRoom(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	occJID, _ := jid.New("room", "conference.jackal.im", "nick", true)
	fullJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	o, err := muc.createOwner(nil, fullJID, occJID)
	require.Nil(t, err)

	roomJID, _ := jid.New("room", "conference.jackal.im", "", true)
	room, err := muc.createRoom(nil, roomJID, o)
	require.Nil(t, err)
	require.NotNil(t, room)
	require.NotNil(t, room.UserToOccupant[*fullJID.ToBareJID()])
	assert.EqualValues(t, room.UserToOccupant[*fullJID.ToBareJID()], *occJID)

	roomMem, err := c.Room().FetchRoom(nil, roomJID)
	require.Nil(t, err)
	require.Equal(t, roomJID.String(), roomMem.RoomJID.String())
}

func TestXEP0045_NewRoom(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)
	err := muc.newRoom(nil, from, to)
	require.Nil(t, err)

	roomMem, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.NotNil(t, roomMem)
	assert.EqualValues(t, to.ToBareJID(), roomMem.RoomJID)
	assert.EqualValues(t, *to, roomMem.UserToOccupant[*from.ToBareJID()])
	require.Equal(t, muc.allRooms[0].String(), to.ToBareJID().String())

	oMem, err := c.Occupant().FetchOccupant(nil, to)
	require.Nil(t, err)
	require.NotNil(t, oMem)
	assert.EqualValues(t, from.ToBareJID(), oMem.BareJID)
}

func TestXEP0045_SendRoomCreateAck(t *testing.T) {
	r, c := setupTest("jackal.im")
	from, _ := jid.New("room", "conference.jackal.im", "nick", true)
	to, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), to)
	stm.SetPresence(xmpp.NewPresence(to.ToBareJID(), to, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	err := muc.sendRoomCreateAck(nil, from, to)
	require.Nil(t, err)
	ack := stm.ReceiveElement()
	require.Equal(t, ack.String(), getAckStanza(from, to).String())
}
