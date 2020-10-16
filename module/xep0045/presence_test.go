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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXEP0045_ExitRoom(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	room, owner := getTestRoomAndOwner(muc)
	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")
	ownerStm := stream.NewMockC2S(uuid.New(), ownerFullJID)
	ownerStm.SetPresence(xmpp.NewPresence(owner.BareJID, ownerFullJID, xmpp.AvailableType))
	r.Bind(context.Background(), ownerStm)

	exUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	exOccJID, _ := jid.New("room", "conference.jackal.im", "temp", true)
	tempOcc := &mucmodel.Occupant{
		OccupantJID: exOccJID,
		BareJID:     exUserJID.ToBareJID(),
		Resources:   map[string]bool{"temp": true},
	}
	muc.repOccupant.UpsertOccupant(nil, tempOcc)
	muc.AddOccupantToRoom(nil, room, tempOcc)

	// presence for exiting the room
	p := xmpp.NewElementName("presence").SetType("unavailable")
	status := xmpp.NewElementName("status").SetText("bye!")
	p.AppendElement(status)
	presence, _ := xmpp.NewPresenceFromElement(p, exUserJID, exOccJID)

	muc.exitRoom(nil, room, presence)

	ack := ownerStm.ReceiveElement()
	require.NotNil(t, ack)
	require.Equal(t, ack.Type(), "unavailable")

	exists, err := muc.repOccupant.OccupantExists(nil, exOccJID)
	require.Nil(t, err)
	require.False(t, exists)

	room, _ = muc.repRoom.FetchRoom(nil, room.RoomJID)
	_, found := room.UserToOccupant[*exUserJID.ToBareJID()]
	require.False(t, found)
}

func TestXEP0045_ChangeStatus(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	room, owner := getTestRoomAndOwner(muc)
	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")
	ownerStm := stream.NewMockC2S(uuid.New(), ownerFullJID)
	ownerStm.SetPresence(xmpp.NewPresence(owner.BareJID, ownerFullJID, xmpp.AvailableType))
	r.Bind(context.Background(), ownerStm)

	stUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	stOccJID, _ := jid.New("room", "conference.jackal.im", "temp", true)
	tempOcc := &mucmodel.Occupant{
		OccupantJID: stOccJID,
		BareJID:     stUserJID.ToBareJID(),
		Resources:   map[string]bool{"temp": true},
	}
	tempOcc.SetAffiliation("admin")
	muc.repOccupant.UpsertOccupant(nil, tempOcc)
	muc.AddOccupantToRoom(nil, room, tempOcc)

	// presence to change the nick
	p := xmpp.NewElementName("presence")
	show := xmpp.NewElementName("show").SetText("xa")
	p.AppendElement(show)
	status := xmpp.NewElementName("status").SetText("my new status")
	p.AppendElement(status)
	presence, _ := xmpp.NewPresenceFromElement(p, stUserJID, stOccJID)
	require.True(t, isChangingStatus(presence))

	muc.changeStatus(nil, room, presence)

	// the user receives status update
	statusStanza := ownerStm.ReceiveElement()
	require.NotNil(t, statusStanza)
	require.NotNil(t, statusStanza.Elements().Child("status"))
	require.NotNil(t, statusStanza.Elements().Child("show"))
}

func TestXEP0045_ChangeNickname(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	room, owner := getTestRoomAndOwner(muc)
	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")

	newOccJID, _ := jid.New("room", "conference.jackal.im", "newnick", true)
	// make sure this nick does not already exist
	occNew, err := muc.repOccupant.FetchOccupant(nil, newOccJID)
	require.Nil(t, err)
	require.Nil(t, occNew)

	// make sure the current nickname exists
	jidBefore, found := room.UserToOccupant[*owner.BareJID]
	require.True(t, found)
	require.Equal(t, jidBefore.String(), owner.OccupantJID.String())
	occBefore, err := muc.repOccupant.FetchOccupant(nil, &jidBefore)
	require.Nil(t, err)
	require.NotNil(t, occBefore)

	ownerStm := stream.NewMockC2S(uuid.New(), ownerFullJID)
	ownerStm.SetPresence(xmpp.NewPresence(owner.BareJID, ownerFullJID, xmpp.AvailableType))
	r.Bind(context.Background(), ownerStm)

	// presence to change the nick
	p := xmpp.NewElementName("presence")
	presence, _ := xmpp.NewPresenceFromElement(p, ownerFullJID, newOccJID)
	require.NotNil(t, presence)

	muc.changeNickname(nil, room, presence)

	// the user receives unavailable stanza
	ackUnavailable := ownerStm.ReceiveElement()
	require.NotNil(t, ackUnavailable)
	require.Equal(t, ackUnavailable.Type(), "unavailable")

	// the user receives presence stanza
	ackPresence := ownerStm.ReceiveElement()
	require.NotNil(t, ackPresence)
	require.Equal(t, ackPresence.From(), newOccJID.String())

	// old nick is deleted
	occBefore, err = muc.repOccupant.FetchOccupant(nil, &jidBefore)
	require.Nil(t, err)
	require.Nil(t, occBefore)

	// new nick is added
	jidAfter, found := room.UserToOccupant[*owner.BareJID]
	require.True(t, found)
	require.Equal(t, jidAfter.String(), newOccJID.String())
	occAfter, err := muc.repOccupant.FetchOccupant(nil, newOccJID)
	require.Nil(t, err)
	require.NotNil(t, occAfter)
	require.Equal(t, occAfter.BareJID.String(), owner.BareJID.String())
}

func TestXEP0045_JoinExistingRoom(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	// existing room
	ownerUserJID, _ := jid.New("milos", "jackal.im", "phone", true)
	ownerOccJID, _ := jid.New("room", "conference.jackal.im", "owner", true)
	owner := &mucmodel.Occupant{
		OccupantJID: ownerOccJID,
		BareJID:     ownerUserJID.ToBareJID(),
		Resources:   map[string]bool{"phone": true},
	}
	owner.SetAffiliation("owner")
	muc.repOccupant.UpsertOccupant(nil, owner)
	ownerStm := stream.NewMockC2S(uuid.New(), ownerUserJID)
	ownerStm.SetPresence(xmpp.NewPresence(ownerUserJID, ownerUserJID, xmpp.AvailableType))
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
	stm.SetPresence(xmpp.NewPresence(from, from, xmpp.AvailableType))
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
	room, owner := getTestRoomAndOwner(muc)

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

func getTestRoomAndOwner(muc *Muc) (*mucmodel.Room, *mucmodel.Occupant) {
	roomConfig := &mucmodel.RoomConfig{
		PwdProtected: true,
		Password:     "secret",
		Open:         false,
		MaxOccCnt:    1,
		AllowInvites: true,
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
	owner := &mucmodel.Occupant{
		OccupantJID: ownerOccJID,
		BareJID:     ownerUserJID.ToBareJID(),
		Resources:   map[string]bool{ownerUserJID.Resource(): true},
	}
	owner.SetAffiliation("owner")

	muc.repOccupant.UpsertOccupant(nil, owner)
	muc.AddOccupantToRoom(context.Background(), room, owner)

	return room, owner
}
