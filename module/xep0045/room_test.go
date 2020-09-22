/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/stream"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
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
	require.NotNil(t, room.UserToOccupant[*fullJID.ToBareJID()])
	assert.EqualValues(t, room.UserToOccupant[*fullJID.ToBareJID()], *occJID)

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

	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	err := muc.sendRoomCreateAck(nil, from, to)
	require.Nil(t, err)
	ack := stm.ReceiveElement()
	require.Equal(t, ack.String(), getAckStanza(from, to).String())
}

func TestXEP0045_RoomAdminsAndOwners(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	rJID, _ := jid.NewWithString("room@conference.jackal.im", true)
	rc := mucmodel.RoomConfig{
		Open: true,
	}
	j1, _ := jid.NewWithString("ortuman@jackal.im", true)
	o1 := &mucmodel.Occupant{
		BareJID:     j1,
		OccupantJID: j1,
	}
	o1.SetAffiliation("")
	muc.repOccupant.UpsertOccupant(context.Background(), o1)

	j2, _ := jid.NewWithString("milos@jackal.im", true)
	o2 := &mucmodel.Occupant{
		BareJID:     j2,
		OccupantJID: j2,
	}
	o2.SetAffiliation("")
	muc.repOccupant.UpsertOccupant(context.Background(), o2)

	occMap := make(map[jid.JID]jid.JID)
	room := &mucmodel.Room{
		RoomJID:        rJID,
		Config:         &rc,
		UserToOccupant: occMap,
	}

	err := muc.AddOccupantToRoom(context.Background(), room, o1)
	require.Nil(t, err)
	err = muc.AddOccupantToRoom(context.Background(), room, o2)
	require.Nil(t, err)

	admins := muc.GetRoomAdmins(context.Background(), room)
	owners := muc.GetRoomOwners(context.Background(), room)
	require.NotNil(t, admins)
	require.Equal(t, len(admins), 0)
	require.NotNil(t, owners)
	require.Equal(t, len(owners), 0)

	muc.SetRoomAdmin(context.Background(), room, o1.OccupantJID)
	admins = muc.GetRoomAdmins(context.Background(), room)
	require.NotNil(t, admins)
	require.Equal(t, len(admins), 1)
	require.Equal(t, admins[0], j1.String())

	muc.SetRoomOwner(context.Background(), room, o2.OccupantJID)
	owners = muc.GetRoomOwners(context.Background(), room)
	require.NotNil(t, owners)
	require.Equal(t, len(owners), 1)
	require.Equal(t, owners[0], j2.String())
}
