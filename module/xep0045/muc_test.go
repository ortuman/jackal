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

	failedMuc := New(&Config{MucHost: "jackal.im"}, nil, c, r)
	require.Nil(t, failedMuc)

	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	require.False(t, muc.router.Hosts().IsConferenceHost("jackal.im"))
	require.True(t, muc.router.Hosts().IsConferenceHost("conference.jackal.im"))

	require.Equal(t, muc.GetMucHostname(), "conference.jackal.im")
}

func TestXEP0045_NewRoomFromPresence(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
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
	//make sure the room is locked
	require.True(t, roomMem.Locked)
}

func TestXEP0045_NewInstantRoomFromIQ(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	// creating a locked room
	err := muc.newRoom(context.Background(), from, to, "room", "nick", true)
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

func TestXEP0045_LegacyGroupchatRoomFromPresence(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	// no <x> element, in order to support legacy groupchat 1.0
	p := xmpp.NewElementName("presence")
	presence, _ := xmpp.NewPresenceFromElement(p, from, to)

	muc.ProcessPresence(context.Background(), presence)

	// sender receives the appropriate response
	ack := stm.ReceiveElement()
	require.Equal(t, ack.String(), getAckStanza(to, from).String())

	// the room is created
	roomMem, _ := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Equal(t, to.ToBareJID().String(), roomMem.RoomJID.String())
	//make sure the room is NOT locked (this is the only difference from MUC)
	require.False(t, roomMem.Locked)
}

func TestXEP0045_NewReservedRoomGetConfig(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	// creating a locked room
	err := muc.newRoom(context.Background(), from, to, "room", "nick", true)
	require.Nil(t, err)
	room, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.True(t, room.Locked)

	// request configuration form
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner)
	iq := xmpp.NewElementName("iq").SetID("create1").SetType("get").AppendElement(query)
	request, err := xmpp.NewIQFromElement(iq, from, to)
	require.Nil(t, err)

	// sending an instant room request into the stream
	require.True(t, muc.MatchesIQ(request))
	require.True(t, isIQForRoomConfigRequest(request))
	muc.ProcessIQ(context.Background(), request)

	// receive the room configuration form
	ack := stm.ReceiveElement()
	require.NotNil(t, ack)
	require.Equal(t, ack.From(), to.String())
	require.Equal(t, ack.To(), from.String())
	require.Equal(t, ack.Name(), "iq")
	require.Equal(t, ack.Type(), "result")
	require.Equal(t, ack.ID(), "create1")

	queryResult := ack.Elements().Child("query")
	require.NotNil(t, queryResult)
	require.Equal(t, queryResult.Namespace(), mucNamespaceOwner)

	formElement := queryResult.Elements().Child("x")
	require.NotNil(t, formElement)
	form, err := xep0004.NewFormFromElement(formElement)
	require.Nil(t, err)
	require.Equal(t, form.Type, xep0004.Form)
	// the total number of fields should be 23
	require.Equal(t, len(form.Fields), 23)
}

func TestXEP0045_NewReservedRoomSubmitConfig(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	from, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	to, _ := jid.New("room", "conference.jackal.im", "nick", true)

	stm := stream.NewMockC2S(uuid.New(), from)
	stm.SetPresence(xmpp.NewPresence(from.ToBareJID(), from, xmpp.AvailableType))
	r.Bind(context.Background(), stm)

	// creating a locked room
	err := muc.newRoom(context.Background(), from, to, "room", "nick", true)
	require.Nil(t, err)
	room, err := muc.reps.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.True(t, room.Locked)
	require.NotEqual(t, room.Name, "Configured Room")

	// get the room configuration form and change the fields
	configForm := muc.getRoomConfigForm(context.Background(), room)
	require.NotNil(t, configForm)
	configForm.Type = xep0004.Submit
	for _, field := range configForm.Fields {
		switch field.Var {
		case ConfigName:
			field.Values = []string{"Configured Room"}
		}
	}

	// generate the form submission IQ stanza
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner)
	query.AppendElement(configForm.Element())
	e := xmpp.NewElementName("iq").SetID("create").SetType("set").AppendElement(query)
	stanza, err := xmpp.NewIQFromElement(e, from, to.ToBareJID())
	require.Nil(t, err)

	// sending the configuration form
	require.True(t, muc.MatchesIQ(stanza))
	require.True(t, isIQForRoomConfigSubmission(stanza))
	muc.ProcessIQ(context.Background(), stanza)

	// receive the response
	ack := stm.ReceiveElement()
	require.NotNil(t, ack)
	//require.Equal(t, ack.Type(), "result")
	//require.Equal(t, ack.Elements().Count(), 0)

	confRoom, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.False(t, confRoom.Locked)
	require.NotEqual(t, confRoom.Name, "Configured Room")
}

func TestModelRoomAdminsAndOwners(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, c, r)
	defer func() { _ = muc.Shutdown() }()

	rJID, _ := jid.NewWithString("room@conference.jackal.im", true)
	rc := mucmodel.RoomConfig{
		Open: true,
	}
	j1, _ := jid.NewWithString("ortuman@jackal.im", true)
	o1 := &mucmodel.Occupant{
		Nick:        "mynick",
		BareJID:     j1,
		OccupantJID: j1,
	}
	o1.SetAffiliation("admin")
	j2, _ := jid.NewWithString("milos@jackal.im", true)
	o2 := &mucmodel.Occupant{
		Nick:        "mynick2",
		BareJID:     j2,
		OccupantJID: j2,
	}
	o2.SetAffiliation("owner")
	occMap := make(map[jid.JID]jid.JID)
	occMap[*o1.BareJID] = *o1.OccupantJID
	occMap[*o2.BareJID] = *o2.OccupantJID

	room := &mucmodel.Room{
		RoomJID:        rJID,
		Config:         &rc,
		UserToOccupant: occMap,
	}

	muc.reps.Occupant().UpsertOccupant(context.Background(), o1)
	muc.reps.Occupant().UpsertOccupant(context.Background(), o2)
	muc.reps.Room().UpsertRoom(context.Background(), room)

	admins := muc.GetRoomAdmins(context.Background(), room)
	owners := muc.GetRoomOwners(context.Background(), room)

	require.NotNil(t, admins)
	require.Equal(t, len(admins), 1)
	require.Equal(t, admins[0], j1.String())

	require.NotNil(t, owners)
	require.Equal(t, len(owners), 1)
	require.Equal(t, owners[0], j2.String())
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
