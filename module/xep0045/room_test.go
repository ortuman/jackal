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
)

func TestXEP0045_CreateRoom(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	occJID, _ := jid.New("room", "conference.jackal.im", "nick", true)
	fullJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	o, err := muc.createOwner(nil, occJID, "nick", fullJID)
	require.Nil(t, err)

	roomJID, _ := jid.New("room", "conference.jackal.im", "", true)
	room, err := muc.createRoom(nil, "testroom", roomJID, o, true)
	require.Nil(t, err)
	require.NotNil(t, room)
	require.Equal(t, room.NickToOccupant["nick"].FullJID.String(), fullJID.String())

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
	err := muc.newRoom(nil, from, to, "room", "nick", false)
	require.Nil(t, err)

	roomMem, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.NotNil(t, roomMem)
	require.Equal(t, to.ToBareJID().String(), roomMem.RoomJID.String())
	require.Equal(t, "nick", roomMem.UserToOccupant[from.ToBareJID().String()].Nick)
	require.Equal(t, muc.allRooms[0].String(), to.ToBareJID().String())

	oMem, err := c.Occupant().FetchOccupant(nil, to)
	require.Nil(t, err)
	require.NotNil(t, oMem)
	require.Equal(t, from.String(), oMem.FullJID.String())
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
