/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"testing"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestXEP0045_JoinExistingRoom(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	// existing room
	ownerUserJID, _ := jid.New("milos", "jackal.im", "phone", true)
	ownerOccJID, _ := jid.New("room", "conference.jackal.im", "owner", true)
	owner := &mucmodel.Occupant{OccupantJID: ownerOccJID, BareJID: ownerUserJID.ToBareJID()}
	owner.SetAffiliation("owner")
	muc.repOccupant.UpsertOccupant(nil, owner)
	ownerStm := stream.NewMockC2S(uuid.New(), ownerUserJID)
	ownerStm.SetPresence(xmpp.NewPresence(ownerUserJID.ToBareJID(), ownerUserJID, xmpp.AvailableType))
	r.Bind(context.Background(), ownerStm)

	roomJID := ownerOccJID.ToBareJID()
	room := &mucmodel.Room{
		Config:         muc.GetDefaultRoomConfig(),
		RoomJID:        roomJID,
		Locked:         false,
		UserToOccupant: make(map[jid.JID]jid.JID),
		InvitedUsers:   make(map[jid.JID]bool),
	}
	room.Config.NonAnonymous = true
	room.Config.PwdProtected = true
	room.Config.Password = "secret"
	room.Config.Open = false
	room.Subject = "Room for testing"
	muc.AddOccupantToRoom(nil, room, owner)

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)
	room.InvitedUsers[*from.ToBareJID()] = true
	muc.repRoom.UpsertRoom(nil, room)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	pwd := xmpp.NewElementName("password").SetText("secret")
	e := xmpp.NewElementNamespace("x", mucNamespace).AppendElement(pwd)
	p := xmpp.NewElementName("presence").AppendElement(e)
	presence, _ := xmpp.NewPresenceFromElement(p, from, to)

	muc.enterRoom(context.Background(), room, presence)

	// sender receives the appropriate response
	ack := stm.ReceiveElement()
	require.NotNil(t, ack)

	// owner receives the appropriate response
	ownerAck := ownerStm.ReceiveElement()
	require.NotNil(t, ownerAck)

	// sender receives the self-presence
	ackSelf := stm.ReceiveElement()
	require.NotNil(t, ackSelf)

	// sender receives the room subject
	ackSubj := stm.ReceiveElement()
	require.NotNil(t, ackSubj)

	// user is in the room
	occ, err := muc.repOccupant.FetchOccupant(context.Background(), to)
	require.Nil(t, err)
	require.NotNil(t, occ)
}

func TestXEP0045_NewRoomRequest(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	e := xmpp.NewElementNamespace("x", mucNamespace)
	p := xmpp.NewElementName("presence").AppendElement(e)
	presence, _ := xmpp.NewPresenceFromElement(p, from, to)

	muc.enterRoom(context.Background(), nil, presence)

	// sender receives the appropriate response
	ack := stm.ReceiveElement()
	require.Equal(t, ack.String(), getAckStanza(to, from).String())

	// the room is created
	roomMem, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.NotNil(t, roomMem)
	require.Equal(t, to.ToBareJID().String(), roomMem.RoomJID.String())
	require.Equal(t, muc.allRooms[0].String(), to.ToBareJID().String())
	oMem, err := c.Occupant().FetchOccupant(nil, to)
	require.Nil(t, err)
	require.NotNil(t, oMem)
	require.Equal(t, from.ToBareJID().String(), oMem.BareJID.String())
	//make sure the room is locked
	require.True(t, roomMem.Locked)
}

func TestXEP0045_OccupantCanEnterRoom(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	// create room and its owner
	room, owner := getRoomAndOwner(muc)

	// owner's c2s stream
	stm := stream.NewMockC2S(uuid.New(), owner.BareJID)
	stm.SetPresence(xmpp.NewPresence(owner.BareJID, owner.BareJID, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	// presence stanza for entering the room correctly
	pwd := xmpp.NewElementName("password").SetText("secret")
	e := xmpp.NewElementNamespace("x", mucNamespace).AppendElement(pwd)
	p := xmpp.NewElementName("presence").AppendElement(e)
	presence, _ := xmpp.NewPresenceFromElement(p, owner.BareJID, owner.OccupantJID)

	// owner can enter
	canEnter, err := muc.occupantCanEnterRoom(context.Background(), room, presence)
	require.Nil(t, err)
	require.True(t, canEnter)

	// lock the room, no one should be able to enter now
	room.Locked = true
	canEnter, err = muc.occupantCanEnterRoom(context.Background(), room, presence)
	require.Nil(t, err)
	require.False(t, canEnter)
	ack := stm.ReceiveElement()
	assert.EqualValues(t, ack, presence.ItemNotFoundError())
	room.Locked = false
}

func getRoomAndOwner(muc *Muc) (*mucmodel.Room, *mucmodel.Occupant) {
	roomConfig := &mucmodel.RoomConfig{
		PwdProtected: true,
		Password:     "secret",
		Open:         false,
		MaxOccCnt:    1,
	}
	roomJID, _ := jid.New("room", "conference.jackal.im", "", true)
	room := &mucmodel.Room{
		Config:         roomConfig,
		RoomJID:        roomJID,
		UserToOccupant: make(map[jid.JID]jid.JID),
		InvitedUsers:   make(map[jid.JID]bool),
	}

	ownerUserJID, _ := jid.New("milos", "jackal.im", "phone", true)
	ownerOccJID, _ := jid.New("room", "conference.jackal.im", "owner", true)
	owner := &mucmodel.Occupant{OccupantJID: ownerOccJID, BareJID: ownerUserJID.ToBareJID()}
	owner.SetAffiliation("owner")

	muc.repOccupant.UpsertOccupant(nil, owner)
	muc.AddOccupantToRoom(context.Background(), room, owner)

	return room, owner
}
