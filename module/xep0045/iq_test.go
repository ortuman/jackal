/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXEP0045_GetOccupantList(t *testing.T) {
	mock := setupTestRoomAndOwner()

	itemEl := xmpp.NewElementName("item").SetAttribute("role", "moderator")
	queryEl := xmpp.NewElementNamespace("query", mucNamespaceAdmin)
	queryEl.AppendElement(itemEl)
	iqEl := xmpp.NewElementName("iq").SetID("admin1").SetType("get")
	iqEl.AppendElement(queryEl)
	iq, _ := xmpp.NewIQFromElement(iqEl, mock.ownerFullJID, mock.room.RoomJID)

	mock.muc.getOccupantList(nil, mock.room, iq)

	resAck := mock.ownerStm.ReceiveElement()
	require.Equal(t, resAck.Type(), "result")
	query := resAck.Elements().Child("query")
	require.NotNil(t, query)
	require.Equal(t, query.Namespace(), mucNamespaceAdmin)
	item := query.Elements().Child("item")
	require.NotNil(t, item)
	require.Equal(t, item.Attributes().Get("nick"), mock.owner.OccupantJID.Resource())
}

func TestXEP0045_ChangeAffiliation(t *testing.T) {
	mock := setupTestRoomAndOwnerAndOcc()
	require.False(t, mock.occ.IsAdmin())

	reasonEl := xmpp.NewElementName("reason").SetText("reason for affiliation change")
	itemEl := xmpp.NewElementName("item").SetAttribute("jid",
		mock.occ.BareJID.String())
	itemEl.SetAttribute("affiliation", "admin").AppendElement(reasonEl)
	queryEl := xmpp.NewElementNamespace("query", mucNamespaceAdmin)
	queryEl.AppendElement(itemEl)
	iqEl := xmpp.NewElementName("iq").SetID("admin1").SetType("set")
	iqEl.AppendElement(queryEl)
	iq, _ := xmpp.NewIQFromElement(iqEl, mock.ownerFullJID, mock.room.RoomJID)

	mock.muc.modifyOccupantList(nil, mock.room, iq)

	acAck := mock.occStm.ReceiveElement()
	require.Equal(t, acAck.From(), mock.occ.OccupantJID.String())
	resArAck := mock.ownerStm.ReceiveElement()
	require.Equal(t, resArAck.From(), mock.occ.OccupantJID.String())
	resAck := mock.ownerStm.ReceiveElement()
	require.Equal(t, resAck.Type(), "result")

	resOcc, _ := mock.muc.repOccupant.FetchOccupant(nil, mock.occ.OccupantJID)
	require.True(t, resOcc.IsAdmin())
}

func TestXEP0045_ChangeRole(t *testing.T) {
	mock := setupTestRoomAndOwnerAndOcc()
	require.False(t, mock.occ.IsModerator())

	reasonEl := xmpp.NewElementName("reason").SetText("reason for role change")
	itemEl := xmpp.NewElementName("item").SetAttribute("nick",
		mock.occ.OccupantJID.Resource())
	itemEl.SetAttribute("role", "moderator").AppendElement(reasonEl)
	queryEl := xmpp.NewElementNamespace("query", mucNamespaceAdmin)
	queryEl.AppendElement(itemEl)
	iqEl := xmpp.NewElementName("iq").SetID("mod1").SetType("set")
	iqEl.AppendElement(queryEl)
	iq, _ := xmpp.NewIQFromElement(iqEl, mock.ownerFullJID, mock.room.RoomJID)

	mock.muc.modifyOccupantList(nil, mock.room, iq)

	rcAck := mock.occStm.ReceiveElement()
	require.Equal(t, rcAck.From(), mock.occ.OccupantJID.String())
	resCrAck := mock.ownerStm.ReceiveElement()
	require.Equal(t, resCrAck.From(), mock.occ.OccupantJID.String())
	resAck := mock.ownerStm.ReceiveElement()
	require.Equal(t, resAck.Type(), "result")

	resOcc, _ := mock.muc.repOccupant.FetchOccupant(nil, mock.occ.OccupantJID)
	require.True(t, resOcc.IsModerator())
}

func TestXEP0045_KickOccupant(t *testing.T) {
	mock := setupTestRoomAndOwnerAndOcc()

	reasonEl := xmpp.NewElementName("reason").SetText("reason for kicking")
	itemEl := xmpp.NewElementName("item").SetAttribute("nick",
		mock.occ.OccupantJID.Resource())
	itemEl.SetAttribute("role", "none").AppendElement(reasonEl)
	queryEl := xmpp.NewElementNamespace("query", mucNamespaceAdmin)
	queryEl.AppendElement(itemEl)
	iqEl := xmpp.NewElementName("iq").SetID("kick1").SetType("set")
	iqEl.AppendElement(queryEl)
	iq, _ := xmpp.NewIQFromElement(iqEl, mock.ownerFullJID, mock.room.RoomJID)

	mock.muc.modifyOccupantList(nil, mock.room, iq)

	kickedAck := mock.occStm.ReceiveElement()
	require.Equal(t, kickedAck.Type(), "unavailable")
	resKickAck := mock.ownerStm.ReceiveElement()
	require.Equal(t, resKickAck.Type(), "unavailable")
	resAck := mock.ownerStm.ReceiveElement()
	require.Equal(t, resAck.Type(), "result")

	_, found := mock.room.GetOccupantJID(mock.occ.BareJID)
	require.False(t, found)
	kicked, _ := mock.muc.repOccupant.FetchOccupant(nil, mock.occ.OccupantJID)
	require.Nil(t, kicked)
}

func TestXEP0045_CreateInstantRoom(t *testing.T) {
	mock := setupTestRoomAndOwner()
	mock.room.Locked = true
	mock.muc.repRoom.UpsertRoom(nil, mock.room)

	// instant room create iq
	x := xmpp.NewElementNamespace("x", xep0004.FormNamespace)
	x.SetAttribute("type", "submit")
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner).AppendElement(x)
	iq := xmpp.NewElementName("iq").SetID("create1").SetType("set")
	iq.AppendElement(query)
	request, _ := xmpp.NewIQFromElement(iq, mock.ownerFullJID, mock.room.RoomJID)

	// sending an instant room request
	require.True(t, isIQForInstantRoomCreate(request))
	mock.muc.createInstantRoom(context.Background(), mock.room, request)

	// receive the instant room creation confirmation
	ack := mock.ownerStm.ReceiveElement()
	require.Equal(t, ack, request.ResultIQ())

	// the room should be unlocked now
	updatedRoom, _ := mock.muc.repRoom.FetchRoom(nil, mock.room.RoomJID)
	require.False(t, updatedRoom.Locked)
}

func TestXEP0045_SendRoomConfiguration(t *testing.T) {
	mock := setupTestRoomAndOwner()
	mock.room.Locked = true
	mock.muc.repRoom.UpsertRoom(nil, mock.room)

	// request configuration form
	query := xmpp.NewElementNamespace("query", mucNamespaceOwner)
	iq := xmpp.NewElementName("iq").SetID("create1").SetType("get")
	iq.AppendElement(query)
	request, _ := xmpp.NewIQFromElement(iq, mock.ownerFullJID, mock.room.RoomJID)

	// sending an instant room request
	require.True(t, mock.muc.MatchesIQ(request))
	require.True(t, isIQForRoomConfigRequest(request))
	mock.muc.sendRoomConfiguration(context.Background(), mock.room, request)

	// receive the room configuration form
	ack := mock.ownerStm.ReceiveElement()
	require.Equal(t, ack.From(), mock.room.RoomJID.String())
	require.Equal(t, ack.To(), mock.ownerFullJID.String())
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
	require.Equal(t, len(form.Fields), 17)
}

func TestXEP0045_ProcessRoomConfiguration(t *testing.T) {
	mock := setupTestRoomAndOwner()
	mock.room.Locked = true
	mock.muc.repRoom.UpsertRoom(nil, mock.room)

	// these fields changed in the configuration
	require.True(t, mock.room.Locked)
	require.NotEqual(t, mock.room.Name, "Configured Room")
	require.NotEqual(t, mock.room.Config.MaxOccCnt, 23)
	require.False(t, mock.room.Config.Public)
	require.False(t, mock.room.Config.NonAnonymous)

	// get the room configuration form and change the fields
	configForm := mock.muc.getRoomConfigForm(context.Background(), mock.room)
	require.NotNil(t, configForm)
	configForm.Type = xep0004.Submit
	for i, field := range configForm.Fields {
		switch field.Var {
		case ConfigName:
			configForm.Fields[i].Values = []string{"Configured Room"}
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
	stanza, err := xmpp.NewIQFromElement(e, mock.ownerFullJID, mock.room.RoomJID)
	require.Nil(t, err)

	// sending the configuration form
	require.True(t, isIQForRoomConfigSubmission(stanza))
	mock.muc.processRoomConfiguration(context.Background(), mock.room, stanza)

	// receive the response
	ack := mock.ownerStm.ReceiveElement()
	assert.EqualValues(t, ack, stanza.ResultIQ())

	// confirm the fields have changed
	confRoom, err := mock.muc.repRoom.FetchRoom(nil, mock.room.RoomJID)
	require.Nil(t, err)
	require.False(t, confRoom.Locked)
	require.Equal(t, confRoom.Name, "Configured Room")
	require.Equal(t, confRoom.Config.MaxOccCnt, 23)
	require.False(t, confRoom.Config.Public)
	require.True(t, confRoom.Config.NonAnonymous)
}
