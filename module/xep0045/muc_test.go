/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0045_NewService(t *testing.T) {
	r, c := setupTest("jackal.im")

	failedMuc := New(&Config{MucHost: "jackal.im"}, nil, r, c.Room(), c.Occupant())
	require.Nil(t, failedMuc)

	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	require.False(t, muc.router.Hosts().IsConferenceHost("jackal.im"))
	require.True(t, muc.router.Hosts().IsConferenceHost("conference.jackal.im"))

	require.Equal(t, muc.GetMucHostname(), "conference.jackal.im")
}

func TestXEP0045_ProcessIQInstantRoom(t *testing.T) {
	mock := setupMockMucService()
	userJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	occupantJID, _ := jid.New("room", "conference.jackal.im", "nick", true)
	stm := stream.NewMockC2S(uuid.New(), userJID)
	stm.SetPresence(xmpp.NewPresence(userJID.ToBareJID(), userJID, xmpp.AvailableType))
	mock.muc.router.Bind(nil, stm)

	// creating a locked room
	err := mock.muc.newRoom(nil, userJID, occupantJID)
	require.Nil(t, err)
	room, err := mock.muc.repRoom.FetchRoom(nil, occupantJID.ToBareJID())
	require.Nil(t, err)
	require.NotNil(t, room)
	require.True(t, room.Locked)

	// instant room create iq
	x := xmpp.NewElementNamespace("x", xep0004.FormNamespace)
	x.SetAttribute("type", "submit")
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner).AppendElement(x)
	iq := xmpp.NewElementName("iq").SetID("create1").SetType("set")
	iq.AppendElement(query)
	request, err := xmpp.NewIQFromElement(iq, userJID, occupantJID)
	require.Nil(t, err)

	// sending an instant room request into the stream
	require.True(t, mock.muc.MatchesIQ(request))
	mock.muc.ProcessIQ(context.Background(), request)

	// receive the instant room creation confirmation
	ack := stm.ReceiveElement()
	require.Equal(t, ack, request.ResultIQ())

	// the room should be unlocked now
	updatedRoom, err := mock.muc.repRoom.FetchRoom(nil, occupantJID.ToBareJID())
	require.False(t, updatedRoom.Locked)
}

func TestXEP0045_ProcessPresenceNewRoom(t *testing.T) {
	mock := setupMockMucService()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	mock.muc.router.Bind(context.Background(), stm)

	e := xmpp.NewElementNamespace("x", mucNamespace)
	p := xmpp.NewElementName("presence").AppendElement(e)
	presence, _ := xmpp.NewPresenceFromElement(p, from, to)

	mock.muc.ProcessPresence(context.Background(), presence)

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

	// the room is locked
	require.True(t, roomMem.Locked)
}

func TestXEP0045_ProcessMessageMsgEveryone(t *testing.T) {
	mock := setupTestRoomAndOwnerAndOcc()

	// owner sends the group message
	body := xmpp.NewElementName("body").SetText("Hello world!")
	msgEl := xmpp.NewMessageType(uuid.New(), "groupchat").AppendElement(body)
	msg, _ := xmpp.NewMessageFromElement(msgEl, mock.ownerFullJID, mock.room.RoomJID)

	mock.muc.ProcessMessage(context.Background(), msg)

	regMsg := mock.occStm.ReceiveElement()
	ownerMsg := mock.ownerStm.ReceiveElement()

	require.Equal(t, regMsg.Type(), "groupchat")
	msgTxt := regMsg.Elements().Child("body").Text()
	require.Equal(t, msgTxt, "Hello world!")

	require.Equal(t, ownerMsg.Type(), "groupchat")
	msgTxt = ownerMsg.Elements().Child("body").Text()
	require.Equal(t, msgTxt, "Hello world!")
}
