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
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/router/host"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const cfgExample = `
host: conference.localhost
name: "Test Server"
`

type mockMucService struct {
	muc          *Muc
	room         *mucmodel.Room
	owner        *mucmodel.Occupant
	ownerFullJID *jid.JID
	ownerStm     *stream.MockC2S
	occ          *mucmodel.Occupant
	occFullJID   *jid.JID
	occStm       *stream.MockC2S
}

func TestXEP0045_MucConfig(t *testing.T) {
	badCfg := `host:`
	cfg := &Config{}
	err := yaml.Unmarshal([]byte(badCfg), &cfg)
	require.NotNil(t, err)

	goodCfg := cfgExample
	cfg = &Config{}
	err = yaml.Unmarshal([]byte(goodCfg), &cfg)
	require.Nil(t, err)
	require.Equal(t, cfg.MucHost, "conference.localhost")
	require.Equal(t, cfg.Name, "Test Server")
	require.NotNil(t, cfg.RoomDefaults)
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

func setupMockMucService() *mockMucService {
	r, rep := setupTest("jackal.im")
	muc := New(&Config{MucHost: "conference.jackal.im"}, nil, r, rep.Room(), rep.Occupant())
	return &mockMucService{muc: muc}
}

func setupTestRoom() *mockMucService {
	mock := setupMockMucService()
	roomConfig := &mucmodel.RoomConfig{
		Open:      true,
		MaxOccCnt: -1,
	}
	roomJID, _ := jid.New("room", "conference.jackal.im", "", true)
	room := &mucmodel.Room{
		Config:  roomConfig,
		RoomJID: roomJID,
	}
	mock.muc.repRoom.UpsertRoom(nil, room)
	mock.room = room
	return mock
}

func setupTestRoomAndOwner() *mockMucService {
	mock := setupTestRoom()

	ownerUserJID, _ := jid.New("milos", "jackal.im", "phone", true)
	ownerOccJID, _ := jid.New("room", "conference.jackal.im", "owner", true)
	owner, _ := mucmodel.NewOccupant(ownerOccJID, ownerUserJID.ToBareJID())
	owner.AddResource(ownerUserJID.Resource())
	owner.SetAffiliation("owner")
	owner.SetRole("moderator")
	mock.muc.AddOccupantToRoom(nil, mock.room, owner)

	ownerStm := stream.NewMockC2S("id-1", ownerUserJID)
	ownerStm.SetPresence(xmpp.NewPresence(owner.BareJID, ownerUserJID, xmpp.AvailableType))
	mock.muc.router.Bind(context.Background(), ownerStm)

	mock.owner = owner
	mock.ownerStm = ownerStm
	mock.ownerFullJID = ownerUserJID
	return mock
}

func setupTestRoomAndOwnerAndOcc() *mockMucService {
	mock := setupTestRoomAndOwner()

	occUserJID, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	occOccJID, _ := jid.New("room", "conference.jackal.im", "occ", true)
	occ, _ := mucmodel.NewOccupant(occOccJID, occUserJID.ToBareJID())
	occ.AddResource(occUserJID.Resource())
	occ.SetAffiliation("")
	occ.SetRole("")
	mock.muc.AddOccupantToRoom(nil, mock.room, occ)

	occStm := stream.NewMockC2S("id-1", occUserJID)
	occStm.SetPresence(xmpp.NewPresence(occ.BareJID, occUserJID, xmpp.AvailableType))
	mock.muc.router.Bind(context.Background(), occStm)

	mock.occ = occ
	mock.occStm = occStm
	mock.occFullJID = occUserJID
	return mock
}
