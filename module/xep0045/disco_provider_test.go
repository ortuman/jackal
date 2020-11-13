/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"context"
	"testing"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestXEP0045_DiscoIdentities(t *testing.T) {
	neRoom, _ := jid.New("nonexistent_room", "conference.jackal.im", "", true)
	sRoom, _ := jid.New("secretroom", "conference.jackal.im", "", true)
	pubRoom, _ := jid.New("publicroom", "conference.jackal.im", "", true)
	usrJID, _ := jid.New("ortuman", "jackal.im", "phone", true)
	cfgJID, _ := jid.New("", "conference.jackal.im", "", true)
	dp := setupDiscoTest()

	ids := dp.Identities(context.Background(), cfgJID, nil, "")
	require.Len(t, ids, 1)
	require.Equal(t, ids[0].Name, dp.service.cfg.Name)

	ids = dp.Identities(context.Background(), neRoom, nil, "")
	require.Len(t, ids, 0)

	ids = dp.Identities(context.Background(), pubRoom, usrJID, mucUserItem)
	require.Len(t, ids, 1)
	require.Equal(t, ids[0].Name, "nick")

	ids = dp.Identities(context.Background(), sRoom, nil, "")
	require.Len(t, ids, 1)
	require.Equal(t, ids[0].Name, "Secret room")
}

func TestXEP0045_DiscoFeatures(t *testing.T) {
	neRoom, _ := jid.New("nonexistent_room", "conference.jackal.im", "", true)
	sRoom, _ := jid.New("secretroom", "conference.jackal.im", "", true)
	dp := setupDiscoTest()

	f, err := dp.Features(context.Background(), nil, nil, "")
	require.Nil(t, err)
	require.Len(t, f, 1)
	require.Equal(t, f[0], mucNamespace)

	f, err = dp.Features(context.Background(), neRoom, nil, "")
	require.Nil(t, f)
	require.NotNil(t, err)

	f, err = dp.Features(context.Background(), sRoom, nil, "")
	require.Nil(t, err)
	require.Len(t, f, 9)
	require.Equal(t, f[3], mucHidden)
}

func TestXEP0045_DiscoItems(t *testing.T) {
	neRoom, _ := jid.New("nonexistent_room", "conference.jackal.im", "", true)
	pRoom, _ := jid.New("publicroom", "conference.jackal.im", "", true)
	dp := setupDiscoTest()

	i, err := dp.Items(context.Background(), nil, nil, "")
	require.Nil(t, err)
	require.Len(t, i, 1)
	require.Equal(t, i[0].Name, "Public room")

	i, err = dp.Items(context.Background(), neRoom, nil, "")
	require.NotNil(t, err)
	require.Nil(t, i)

	i, err = dp.Items(context.Background(), pRoom, nil, "")
	require.Nil(t, err)
	require.NotNil(t, i)
	require.Len(t, i, 1)
	require.Equal(t, i[0].Jid, "publicroom@conference.jackal.im/nick")
}

func setupDiscoTest() *discoMucProvider {
	mock := setupMockMucService()
	mock.muc.cfg.Name = "Chat Service"

	hiddenRc := &mucmodel.RoomConfig{Public: false}
	hJID, _ := jid.New("secretroom", "conference.jackal.im", "", true)
	hiddenRoom := mucmodel.Room{
		Name:    "Secret room",
		Config:  hiddenRc,
		RoomJID: hJID,
	}

	publicRc := &mucmodel.RoomConfig{Public: true}
	pJID, _ := jid.New("publicroom", "conference.jackal.im", "", true)
	publicRoom := mucmodel.Room{
		Name:           "Public room",
		Config:         publicRc,
		RoomJID:        pJID,
	}
	publicRoom.Config.SetWhoCanGetMemberList("all")
	oJID, _ := jid.New("publicroom", "conference.jackal.im", "nick", true)
	usrJID, _ := jid.New("ortuman", "jackal.im", "phone", true)
	o := &mucmodel.Occupant{OccupantJID: oJID, BareJID: usrJID.ToBareJID()}
	publicRoom.AddOccupant(o)

	mock.muc.repRoom.UpsertRoom(context.Background(), &publicRoom)
	mock.muc.repRoom.UpsertRoom(context.Background(), &hiddenRoom)
	mock.muc.allRooms = append(mock.muc.allRooms, *hiddenRoom.RoomJID)
	mock.muc.allRooms = append(mock.muc.allRooms, *publicRoom.RoomJID)

	return &discoMucProvider{service: mock.muc}
}
