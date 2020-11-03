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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXEP0045_ExitRoom(t *testing.T) {
	mock := setupTestRoomAndOwnerAndOcc()

	// presence for exiting the room
	p := xmpp.NewElementName("presence").SetType("unavailable")
	status := xmpp.NewElementName("status").SetText("bye!")
	p.AppendElement(status)
	presence, _ := xmpp.NewPresenceFromElement(p, mock.occFullJID, mock.occ.OccupantJID)

	mock.muc.exitRoom(nil, mock.room, presence)

	ack := mock.ownerStm.ReceiveElement()
	require.Equal(t, ack.Type(), "unavailable")

	exists, err := mock.muc.repOccupant.OccupantExists(nil, mock.occ.OccupantJID)
	require.Nil(t, err)
	require.False(t, exists)

	room, _ := mock.muc.repRoom.FetchRoom(nil, mock.room.RoomJID)
	require.NotNil(t, room)
	require.False(t, room.UserIsInRoom(mock.occ.BareJID))
}

func TestXEP0045_ChangeStatus(t *testing.T) {
	mock := setupTestRoomAndOwnerAndOcc()

	// presence to change the nick
	p := xmpp.NewElementName("presence")
	show := xmpp.NewElementName("show").SetText("xa")
	p.AppendElement(show)
	status := xmpp.NewElementName("status").SetText("my new status")
	p.AppendElement(status)
	presence, _ := xmpp.NewPresenceFromElement(p, mock.ownerFullJID, mock.owner.OccupantJID)
	require.True(t, isChangingStatus(presence))

	mock.muc.changeStatus(nil, mock.room, presence)

	// the user receives status update
	statusStanza := mock.occStm.ReceiveElement()
	require.NotNil(t, statusStanza)
	require.Equal(t, statusStanza.Elements().Child("status").Text(), "my new status")
	require.NotNil(t, statusStanza.Elements().Child("show"))
}

func TestXEP0045_ChangeNickname(t *testing.T) {
	mock := setupTestRoomAndOwnerAndOcc()
	newOccJID, _ := jid.New("room", "conference.jackal.im", "newnick", true)

	// presence to change the nick
	p := xmpp.NewElementName("presence")
	presence, _ := xmpp.NewPresenceFromElement(p, mock.ownerFullJID, newOccJID)
	require.NotNil(t, presence)

	mock.muc.changeNickname(nil, mock.room, presence)

	// the user receives unavailable stanza
	ackUnavailable := mock.occStm.ReceiveElement()
	require.NotNil(t, ackUnavailable)
	require.Equal(t, ackUnavailable.Type(), "unavailable")

	// the user receives presence stanza
	ackPresence := mock.occStm.ReceiveElement()
	require.NotNil(t, ackPresence)
	require.Equal(t, ackPresence.From(), newOccJID.String())

	// old nick is deleted
	occBefore, err := mock.muc.repOccupant.FetchOccupant(nil, mock.owner.OccupantJID)
	require.Nil(t, err)
	require.Nil(t, occBefore)

	// new nick is added
	jidAfter, _ := mock.room.GetOccupantJID(mock.owner.BareJID)
	require.NotNil(t, jidAfter)
	require.Equal(t, jidAfter.String(), newOccJID.String())
	occAfter, err := mock.muc.repOccupant.FetchOccupant(nil, newOccJID)
	require.Nil(t, err)
	require.NotNil(t, occAfter)
	require.Equal(t, occAfter.BareJID.String(), mock.owner.BareJID.String())
}

func TestXEP0045_JoinExistingRoom(t *testing.T) {
	mock := setupTestRoomAndOwner()
	mock.room.Config.PwdProtected = true
	mock.room.Config.Password = "secret"
	mock.room.Config.Open = false
	mock.room.Subject = "Room for testing"

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)
	mock.room.InviteUser(from.ToBareJID())
	mock.muc.repRoom.UpsertRoom(nil, mock.room)

	newStm := stream.NewMockC2S(uuid.New(), from)
	newStm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	mock.muc.router.Bind(context.Background(), newStm)

	pwd := xmpp.NewElementName("password").SetText("secret")
	e := xmpp.NewElementNamespace("x", mucNamespace).AppendElement(pwd)
	p := xmpp.NewElementName("presence").AppendElement(e)
	presence, _ := xmpp.NewPresenceFromElement(p, from, to)

	mock.muc.enterRoom(context.Background(), mock.room, presence)

	// sender receives the appropriate response
	ack := newStm.ReceiveElement()
	require.Equal(t, ack.From(), mock.owner.OccupantJID.String())

	// owner receives the appropriate response
	ownerAck := mock.ownerStm.ReceiveElement()
	require.Equal(t, ownerAck.From(), to.String())

	// sender receives the self-presence
	ackSelf := newStm.ReceiveElement()
	require.Equal(t, ackSelf.From(), to.String())

	// sender receives the room subject
	ackSubj := newStm.ReceiveElement()
	require.NotNil(t, ackSubj.Elements().Child("subject").Text(), "Room for testing")

	// user is in the room
	occ, err := mock.muc.repOccupant.FetchOccupant(context.Background(), to)
	require.Nil(t, err)
	require.NotNil(t, occ)
}

func TestXEP0045_NewRoomRequest(t *testing.T) {
	mock := setupMockMucService()
	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	mock.muc.router.Bind(context.Background(), stm)

	e := xmpp.NewElementNamespace("x", mucNamespace)
	p := xmpp.NewElementName("presence").AppendElement(e)
	presence, _ := xmpp.NewPresenceFromElement(p, from, to)

	mock.muc.enterRoom(context.Background(), nil, presence)

	// sender receives the appropriate response
	ack := stm.ReceiveElement()
	require.Equal(t, ack.String(), getAckStanza(to, from).String())

	// the room is created
	roomMem, err := mock.muc.repRoom.FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.NotNil(t, roomMem)
	require.Equal(t, to.ToBareJID().String(), roomMem.RoomJID.String())
	require.Equal(t, mock.muc.allRooms[0].String(), to.ToBareJID().String())
	oMem, err := mock.muc.repOccupant.FetchOccupant(nil, to)
	require.Nil(t, err)
	require.NotNil(t, oMem)
	require.Equal(t, from.ToBareJID().String(), oMem.BareJID.String())
	//make sure the room is locked
	require.True(t, roomMem.Locked)
}

func TestXEP0045_OccupantCanEnterRoom(t *testing.T) {
	mock := setupTestRoomAndOwner()

	// presence stanza for entering the room correctly
	pwd := xmpp.NewElementName("password").SetText("secret")
	e := xmpp.NewElementNamespace("x", mucNamespace).AppendElement(pwd)
	p := xmpp.NewElementName("presence").AppendElement(e)
	presence, _ := xmpp.NewPresenceFromElement(p, mock.ownerFullJID, mock.owner.OccupantJID)

	// owner can enter
	canEnter, err := mock.muc.occupantCanEnterRoom(context.Background(), mock.room, presence)
	require.Nil(t, err)
	require.True(t, canEnter)

	// lock the room, no one should be able to enter now
	mock.room.Locked = true
	canEnter, err = mock.muc.occupantCanEnterRoom(context.Background(), mock.room, presence)
	require.Nil(t, err)
	require.False(t, canEnter)
	ack := mock.ownerStm.ReceiveElement()
	assert.EqualValues(t, ack, presence.ItemNotFoundError())
	room, _ := mock.muc.repRoom.FetchRoom(nil, mock.room.RoomJID)
	room.Locked = false
}
