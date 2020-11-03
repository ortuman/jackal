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

func TestXEP0045_ChangeSubject(t *testing.T) {
	mock := setupTestRoomAndOwner()

	subjectEl := xmpp.NewElementName("subject").SetText("new subject")
	msgEl := xmpp.NewElementName("message").SetType("groupchat")
	msgEl.AppendElement(subjectEl)
	msg, _ := xmpp.NewMessageFromElement(msgEl, mock.ownerFullJID, mock.room.RoomJID)

	mock.muc.changeSubject(nil, mock.room, msg)

	ack := mock.ownerStm.ReceiveElement()
	require.Equal(t, ack.Type(), "groupchat")
	newSubject := ack.Elements().Child("subject")
	require.NotNil(t, newSubject)
	require.Equal(t, newSubject.Text(), "new subject")

	updatedRoom, _ := mock.muc.repRoom.FetchRoom(nil, mock.room.RoomJID)
	require.Equal(t, updatedRoom.Subject, "new subject")
}

func TestXEP0045_DeclineInvite(t *testing.T) {
	mock := setupTestRoomAndOwner()
	invitedUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	mock.room.InviteUser(invitedUserJID.ToBareJID())
	mock.muc.repRoom.UpsertRoom(nil, mock.room)

	// user declines the invitation
	reason := xmpp.NewElementName("reason").SetText("Sorry, not for me!")
	invite := xmpp.NewElementName("decline")
	invite.SetAttribute("to", mock.owner.BareJID.String()).AppendElement(reason)
	x := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(invite)
	m := xmpp.NewElementName("message").SetID("id-decline").AppendElement(x)
	msg, _ := xmpp.NewMessageFromElement(m, invitedUserJID, mock.room.RoomJID)

	require.True(t, isDeclineInvitation(msg))
	mock.muc.declineInvitation(nil, mock.room, msg)

	decline := mock.ownerStm.ReceiveElement()
	require.Equal(t, decline.From(), mock.room.RoomJID.String())
	room, _ := mock.muc.repRoom.FetchRoom(nil, mock.room.RoomJID)
	require.False(t, room.UserIsInvited(invitedUserJID.ToBareJID()))
}

func TestXEP0045_SendInvite(t *testing.T) {
	mock := setupTestRoomAndOwner()
	mock.room.Config.AllowInvites = true
	mock.muc.repRoom.UpsertRoom(nil, mock.room)

	invitedUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	invStm := stream.NewMockC2S("id-2", invitedUserJID)
	invStm.SetPresence(xmpp.NewPresence(invitedUserJID.ToBareJID(), invitedUserJID,
		xmpp.AvailableType))
	mock.muc.router.Bind(context.Background(), invStm)

	// user is not already invited
	require.False(t, mock.room.UserIsInvited(invitedUserJID.ToBareJID()))

	// owner sends the invitation
	reason := xmpp.NewElementName("reason").SetText("Join me!")
	invite := xmpp.NewElementName("invite")
	invite.SetAttribute("to", invitedUserJID.ToBareJID().String())
	invite.AppendElement(reason)
	x := xmpp.NewElementNamespace("x", mucNamespaceUser).AppendElement(invite)
	m := xmpp.NewElementName("message").SetID("id-invite").AppendElement(x)
	msg, err := xmpp.NewMessageFromElement(m, mock.ownerFullJID, mock.room.RoomJID)
	require.Nil(t, err)

	require.True(t, isInvite(msg))
	mock.muc.inviteUser(context.Background(), mock.room, msg)

	inviteStanza := invStm.ReceiveElement()
	require.Equal(t, inviteStanza.From(), mock.room.RoomJID.String())

	updatedRoom, _ := mock.muc.repRoom.FetchRoom(nil, mock.room.RoomJID)
	require.True(t, updatedRoom.UserIsInvited(invitedUserJID.ToBareJID()))
}

func TestXEP0045_MessageEveryone(t *testing.T) {
	mock := setupTestRoomAndOwnerAndOcc()

	// owner sends the group message
	body := xmpp.NewElementName("body").SetText("Hello world!")
	msgEl := xmpp.NewMessageType(uuid.New(), "groupchat").AppendElement(body)
	msg, _ := xmpp.NewMessageFromElement(msgEl, mock.ownerFullJID, mock.room.RoomJID)

	mock.muc.messageEveryone(context.Background(), mock.room, msg)

	regMsg := mock.occStm.ReceiveElement()
	ownerMsg := mock.ownerStm.ReceiveElement()

	require.Equal(t, regMsg.Type(), "groupchat")
	msgTxt := regMsg.Elements().Child("body").Text()
	require.Equal(t, msgTxt, "Hello world!")

	require.Equal(t, ownerMsg.Type(), "groupchat")
	msgTxt = ownerMsg.Elements().Child("body").Text()
	require.Equal(t, msgTxt, "Hello world!")
}

func TestXEP0045_SendPM(t *testing.T) {
	mock := setupTestRoomAndOwnerAndOcc()
	mock.room.Config.SetWhoCanSendPM("all")
	mock.muc.repRoom.UpsertRoom(nil, mock.room)

	// owner sends the private message
	body := xmpp.NewElementName("body").SetText("Hello ortuman!")
	msgEl := xmpp.NewMessageType(uuid.New(), "chat").AppendElement(body)
	m, _ := xmpp.NewMessageFromElement(msgEl, mock.ownerFullJID, mock.occ.OccupantJID)

	mock.muc.sendPM(context.Background(), mock.room, m)

	regMsg := mock.occStm.ReceiveElement()
	require.Equal(t, regMsg.Type(), "chat")
	msgTxt := regMsg.Elements().Child("body").Text()
	require.Equal(t, msgTxt, "Hello ortuman!")
}
