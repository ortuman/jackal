/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"testing"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestXEP0045_KickOccupant(t *testing.T) {
	r, c := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, c.Room(), c.Occupant())
	defer func() { _ = muc.Shutdown() }()

	room, owner := getTestRoomAndOwner(muc)
	ownerFullJID := addResourceToBareJID(owner.BareJID, "phone")

	ownerStm := stream.NewMockC2S("id-1", ownerFullJID)
	ownerStm.SetPresence(xmpp.NewPresence(owner.BareJID, ownerFullJID, xmpp.AvailableType))
	r.Bind(context.Background(), ownerStm)

	kickedUsrJID, _ := jid.New("to_be_kicked", "jackal.im", "office", true)
	kickedOccJID, _ := jid.New("room", "conference.jackal.im", "kicked", true)
	kickedOcc, err := mucmodel.NewOccupant(kickedOccJID, kickedUsrJID.ToBareJID())
	require.Nil(t, err)
	kickedOcc.AddResource("office")
	muc.AddOccupantToRoom(nil, room, kickedOcc)

	kickedStm := stream.NewMockC2S("id-1", kickedUsrJID)
	kickedStm.SetPresence(xmpp.NewPresence(kickedOcc.BareJID, kickedUsrJID, xmpp.AvailableType))
	r.Bind(context.Background(), kickedStm)

	reasonEl := xmpp.NewElementName("reason").SetText("reason for kicking")
	itemEl := xmpp.NewElementName("item").SetAttribute("nick", kickedOccJID.Resource())
	itemEl.SetAttribute("role", "none").AppendElement(reasonEl)
	queryEl := xmpp.NewElementNamespace("query", mucNamespaceAdmin).AppendElement(itemEl)
	iqEl := xmpp.NewElementName("iq").SetID("kick1").SetType("set").AppendElement(queryEl)
	iq, err := xmpp.NewIQFromElement(iqEl, ownerFullJID, room.RoomJID)
	require.True(t, isIQForKickOccupant(iq))

	muc.kickOccupant(nil, room, iq)

	kickedAck := kickedStm.ReceiveElement()
	require.Equal(t, kickedAck.Type(), "unavailable")
	resAck := ownerStm.ReceiveElement()
	require.Equal(t, resAck.Type(), "result")
	resKickAck := ownerStm.ReceiveElement()
	require.Equal(t, resKickAck.Type(), "unavailable")

	_, found := room.GetOccupantJID(kickedUsrJID.ToBareJID())
	require.False(t, found)
	kicked, _ := muc.repOccupant.FetchOccupant(nil, kickedOccJID)
	require.Nil(t, kicked)
}

func TestXEP0045_CreateInstantRoom(t *testing.T) {
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
	require.True(t, isIQForInstantRoomCreate(request))
	muc.createInstantRoom(context.Background(), room, request)
	//muc.ProcessIQ(context.Background(), request)

	// receive the instant room creation confirmation
	ack := stm.ReceiveElement()
	require.NotNil(t, ack)
	require.Equal(t, ack, request.ResultIQ())

	// the room should be unlocked now
	updatedRoom, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.False(t, updatedRoom.Locked)
}

func TestXEP0045_SendRoomConfiguration(t *testing.T) {
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

	// request configuration form
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner)
	iq := xmpp.NewElementName("iq").SetID("create1").SetType("get").AppendElement(query)
	request, err := xmpp.NewIQFromElement(iq, from, to)
	require.Nil(t, err)

	// sending an instant room request into the stream
	require.True(t, muc.MatchesIQ(request))
	require.True(t, isIQForRoomConfigRequest(request))
	muc.sendRoomConfiguration(context.Background(), room, request)

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
	// the total number of fields should be 20
	require.Equal(t, len(form.Fields), 19)
}

func TestXEP0045_ProcessRoomConfiguration(t *testing.T) {
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
	room, err := muc.repRoom.FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)

	// these fields changed in the configuration
	require.True(t, room.Locked)
	require.NotEqual(t, room.Name, "Configured Room")
	require.NotEqual(t, room.Name, "Configured Room")
	require.NotEqual(t, room.Config.MaxOccCnt, 23)
	require.False(t, room.Config.Public)
	require.False(t, room.Config.NonAnonymous)

	// occupant to be promoted into an admin
	milosJID, _ := jid.New("milos", "jackal.im", "office", true)
	occJID, _ := jid.New("room", "conference.jackal.im", "milos", true)
	o, _ := mucmodel.NewOccupant(occJID, milosJID.ToBareJID())
	o.AddResource("office")
	muc.repOccupant.UpsertOccupant(context.Background(), o)
	room.AddOccupant(o)
	muc.repRoom.UpsertRoom(context.Background(), room)
	require.False(t, o.IsAdmin())

	// get the room configuration form and change the fields
	configForm := muc.getRoomConfigForm(context.Background(), room)
	require.NotNil(t, configForm)
	configForm.Type = xep0004.Submit
	for i, field := range configForm.Fields {
		switch field.Var {
		case ConfigName:
			configForm.Fields[i].Values = []string{"Configured Room"}
		case ConfigAdmins:
			configForm.Fields[i].Values = []string{milosJID.ToBareJID().String()}
		case ConfigMaxUsers:
			configForm.Fields[i].Values = []string{"23"}
		case ConfigWhoIs:
			configForm.Fields[i].Values = []string{"1"}
		case ConfigPublic:
			configForm.Fields[i].Values = []string{"0"}
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
	muc.processRoomConfiguration(context.Background(), room, stanza)

	// receive the response
	ack := stm.ReceiveElement()
	require.NotNil(t, ack)
	assert.EqualValues(t, ack, stanza.ResultIQ())

	// confirm the fields have changed
	confRoom, err := c.Room().FetchRoom(nil, to.ToBareJID())
	require.Nil(t, err)
	require.False(t, confRoom.Locked)
	require.Equal(t, confRoom.Name, "Configured Room")
	require.Equal(t, confRoom.Config.MaxOccCnt, 23)
	require.False(t, confRoom.Config.Public)
	require.True(t, confRoom.Config.NonAnonymous)

	// occupant got promoted to admin
	updatedOcc, err := c.Occupant().FetchOccupant(context.Background(), occJID)
	require.Nil(t, err)
	require.NotNil(t, updatedOcc)
	require.True(t, updatedOcc.IsAdmin())
}
