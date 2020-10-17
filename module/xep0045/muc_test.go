/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"crypto/tls"
	"testing"

	c2srouter "github.com/ortuman/jackal/c2s/router"
	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/router/host"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/repository"
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
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	// creating a locked room
	err := muc.newRoom(context.Background(), from, to)
	require.Nil(t, err)
	room, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.True(t, room.Locked)

	// instant room create iq
	x := xmpp.NewElementNamespace("x", xep0004.FormNamespace).SetAttribute("type", "submit")
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner).AppendElement(x)
	iq := xmpp.NewElementName("iq").SetID("create1").SetType("set").AppendElement(query)
	request, err := xmpp.NewIQFromElement(iq, from, to)
	require.Nil(t, err)

	// sending an instant room request into the stream
	require.True(t, muc.MatchesIQ(request))
	muc.ProcessIQ(context.Background(), request)

	// receive the instant room creation confirmation
	ack := stm.ReceiveElement()
	require.NotNil(t, ack)
	require.Equal(t, ack, request.ResultIQ())

	// the room should be unlocked now
	updatedRoom, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.False(t, updatedRoom.Locked)
}

func TestXEP0045_ProcessPresenceNewRoom(t *testing.T) {
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

	muc.ProcessPresence(context.Background(), presence)

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

	// the room is locked
	require.True(t, roomMem.Locked)
}

func TestXEP0045_ProcessMessageMsgEveryone(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	room, owner := getTestRoomAndOwner(muc)
	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")

	regularUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	regularOccJID, _ := jid.New("room", "conference.jackal.im", "ort", true)
	regularOcc, _ := mucmodel.NewOccupant(regularOccJID, regularUserJID.ToBareJID())
	regularOcc.AddResource(regularUserJID.Resource())
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

	muc.ProcessMessage(context.Background(), msg)

	regMsg := regStm.ReceiveElement()
	ownerMsg := ownerStm.ReceiveElement()

	require.Equal(t, regMsg.Type(), "groupchat")
	msgTxt := regMsg.Elements().Child("body").Text()
	require.Equal(t, msgTxt, "Hello world!")

	require.Equal(t, ownerMsg.Type(), "groupchat")
	msgTxt = ownerMsg.Elements().Child("body").Text()
	require.Equal(t, msgTxt, "Hello world!")
}

func setupTest(domain string) (router.Router, repository.Container) {
	hosts, _ := host.New([]host.Config{{Name: domain, Certificate: tls.Certificate{}}})
	rep, _ := memorystorage.New()
	r, _ := router.New(
		hosts,
		c2srouter.New(rep.User(), memorystorage.NewBlockList()),
		nil,
	)
	return r, rep
}
