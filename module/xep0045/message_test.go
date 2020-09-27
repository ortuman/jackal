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

func TestXEP0045_MessageEveryone(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	room, owner := getTestRoomAndOwner(muc)
	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")

	regularUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	regularOccJID, _ := jid.New("room", "conference.jackal.im", "ort", true)
	regularOcc := &mucmodel.Occupant{
		OccupantJID: regularOccJID,
		BareJID:     regularUserJID.ToBareJID(),
		Resources:   map[string]bool{regularUserJID.Resource(): true},
	}
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
	regularOcc := &mucmodel.Occupant{
		OccupantJID: regularOccJID,
		BareJID:     regularUserJID.ToBareJID(),
		Resources:   map[string]bool{regularUserJID.Resource(): true},
	}
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
