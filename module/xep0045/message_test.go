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
)
func TestXEP0045_DeclineInvite(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	room, owner := getTestRoomAndOwner(muc)
	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")
	regularUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	room.InvitedUsers[*regularUserJID.ToBareJID()] = true
	muc.repRoom.UpsertRoom(nil, room)

	ownerStm := stream.NewMockC2S("id-1", ownerFullJID)
	ownerStm.SetPresence(xmpp.NewPresence(owner.BareJID, ownerFullJID, xmpp.AvailableType))
	r.Bind(context.Background(), ownerStm)

	// user declines the invitation
	reason := xmpp.NewElementName("reason").SetText("Sorry, not for me!")
	invite := xmpp.NewElementName("decline").SetAttribute("to", owner.BareJID.String())
	invite.AppendElement(reason)
	x := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(invite)
	m := xmpp.NewElementName("message").SetID("id-decline").AppendElement(x)
	msg, err := xmpp.NewMessageFromElement(m, regularUserJID, room.RoomJID)
	require.Nil(t, err)

	require.True(t, isDeclineInvitation(msg))
	muc.declineInvitation(nil, room, msg)

	decline := ownerStm.ReceiveElement()
	require.Equal(t, decline.From(), room.RoomJID.String())
	room, _ = muc.repRoom.FetchRoom(nil, room.RoomJID)
	_, found := room.InvitedUsers[*regularUserJID.ToBareJID()]
	require.False(t, found)
}

func TestXEP0045_SendInvite(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	room, owner := getTestRoomAndOwner(muc)
	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")

	regularUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	regStm := stream.NewMockC2S("id-2", regularUserJID)
	regStm.SetPresence(xmpp.NewPresence(regularUserJID.ToBareJID(), regularUserJID, xmpp.AvailableType))
	r.Bind(context.Background(), regStm)

	// make sure user is not already invited
	_, found := room.InvitedUsers[*regularUserJID.ToBareJID()]
	require.False(t, found)

	// owner sends the invitation
	reason := xmpp.NewElementName("reason").SetText("Join me!")
	invite := xmpp.NewElementName("invite").SetAttribute("to", regularUserJID.ToBareJID().String())
	invite.AppendElement(reason)
	x := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(invite)
	m := xmpp.NewElementName("message").SetID("id-invite").AppendElement(x)
	msg, err := xmpp.NewMessageFromElement(m, ownerFullJID, room.RoomJID)
	require.Nil(t, err)

	require.True(t, isInvite(msg))
	muc.inviteUser(context.Background(), room, msg)

	inviteStanza := regStm.ReceiveElement()
	require.Equal(t, inviteStanza.From(), room.RoomJID.String())

	updatedRoom, _ := muc.repRoom.FetchRoom(nil, room.RoomJID)
	_, found = updatedRoom.InvitedUsers[*regularUserJID.ToBareJID()]
	require.True(t, found)
}

func TestXEP0045_MessageEveryone(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	room, owner := getTestRoomAndOwner(muc)
	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")

	regularUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	regularOccJID, _ := jid.New("room", "conference.jackal.im", "ort", true)
	regularOcc, err := mucmodel.NewOccupant(regularOccJID, regularUserJID.ToBareJID())
	regularOcc.AddResource("balcony")
	muc.repOccupant.UpsertOccupant(nil, regularOcc)
	muc.AddOccupantToRoom(nil, room, regularOcc)

	ownerStm := stream.NewMockC2S("id-1", ownerFullJID)
	regStm := stream.NewMockC2S("id-2", regularUserJID)

	ownerStm.SetPresence(xmpp.NewPresence(owner.BareJID, ownerFullJID, xmpp.AvailableType))
	regStm.SetPresence(xmpp.NewPresence(regularOcc.BareJID, regularUserJID, xmpp.AvailableType))

	r.Bind(context.Background(), ownerStm)
	r.Bind(context.Background(), regStm)

	// owner sends the group message
	body := xmpp.NewElementName("body").SetText("Hello world!")
	msgEl := xmpp.NewMessageType(uuid.New(), "groupchat").AppendElement(body)
	msg, err := xmpp.NewMessageFromElement(msgEl, ownerFullJID, room.RoomJID)
	require.Nil(t, err)

	muc.messageEveryone(context.Background(), room, msg)

	regMsg := regStm.ReceiveElement()
	ownerMsg := ownerStm.ReceiveElement()

	require.Equal(t, regMsg.Type(), "groupchat")
	msgTxt := regMsg.Elements().Child("body").Text()
	require.Equal(t, msgTxt, "Hello world!")

	require.Equal(t, ownerMsg.Type(), "groupchat")
	msgTxt = ownerMsg.Elements().Child("body").Text()
	require.Equal(t, msgTxt, "Hello world!")
}

func TestXEP0045_SendPM(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	room, owner := getTestRoomAndOwner(muc)
	room.Config.SetWhoCanSendPM("all")
	muc.repRoom.UpsertRoom(nil, room)

	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")

	regularUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	regularOccJID, _ := jid.New("room", "conference.jackal.im", "ort", true)
	regularOcc, _ := mucmodel.NewOccupant(regularOccJID, regularUserJID.ToBareJID())
	regularOcc.AddResource(regularUserJID.Resource())
	muc.repOccupant.UpsertOccupant(nil, regularOcc)
	muc.AddOccupantToRoom(nil, room, regularOcc)

	regStm := stream.NewMockC2S("id-2", regularUserJID)
	regStm.SetPresence(xmpp.NewPresence(regularOcc.BareJID, regularUserJID, xmpp.AvailableType))
	r.Bind(context.Background(), regStm)

	// owner sends the private message
	body := xmpp.NewElementName("body").SetText("Hello ortuman!")
	msgEl := xmpp.NewMessageType(uuid.New(), "chat").AppendElement(body)
	msg, err := xmpp.NewMessageFromElement(msgEl, ownerFullJID, regularOccJID)
	require.Nil(t, err)

	muc.sendPM(context.Background(), room, msg)

	regMsg := regStm.ReceiveElement()
	require.Equal(t, regMsg.Type(), "chat")
	msgTxt := regMsg.Elements().Child("body").Text()
	require.Equal(t, msgTxt, "Hello ortuman!")
}

func TestXEP0045_MessageOccupant(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	_, owner := getTestRoomAndOwner(muc)
	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")
	ownerStm := stream.NewMockC2S("id-1", ownerFullJID)
	ownerStm.SetPresence(xmpp.NewPresence(owner.BareJID, ownerFullJID, xmpp.AvailableType))
	r.Bind(context.Background(), ownerStm)

	senderJID, _ := jid.New("sender", "jackal.im", "phone", false)
	body := xmpp.NewElementName("body").SetText("hello")

	muc.messageOccupant(nil, owner.OccupantJID, senderJID, body, uuid.New(), true)

	msg := ownerStm.ReceiveElement()
	require.Equal(t, "hello", msg.Elements().Child("body").Text())
}
