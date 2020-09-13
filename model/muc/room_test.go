/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ortuman/jackal/xmpp/jid"
)

func TestModelRoom(t *testing.T){
	rJID, _ := jid.NewWithString("room@conference.jackal.im", true)
	rc := RoomConfig{
		Open: true,
	}
	jFull, _ := jid.NewWithString("ortuman@jackal.im/laptop", true)
	o := &Occupant{
		Nick: "mynick",
		FullJID: jFull,
		OccupantJID: jFull,
	}
	occMap := make(map[string]*Occupant)
	occMap[o.Nick] = o
	userMap := make(map[string]*Occupant)
	userMap[o.FullJID.ToBareJID().String()] = o

	r1 := Room{
		Name: "Test Room",
		RoomJID: rJID,
		Desc: "Test Description",
		Language: "eng",
		Config: &rc,
		OccupantsCnt: 1,
		NickToOccupant: occMap,
		UserToOccupant: userMap,
		Locked: true,
	}

	buf := new(bytes.Buffer)
	require.Nil(t, r1.ToBytes(buf))

	r2 := Room{}
	require.Nil(t, r2.FromBytes(buf))
	assert.EqualValues(t, r1, r2)
}

func TestModelRoomAdminsAndOwners(t *testing.T){
	rJID, _ := jid.NewWithString("room@conference.jackal.im", true)
	rc := RoomConfig{
		Open: true,
	}
	j1, _ := jid.NewWithString("ortuman@jackal.im", true)
	o1 := &Occupant{
		Nick: "mynick",
		FullJID: j1,
		OccupantJID: j1,
		affiliation: "admin",
	}
	j2, _ := jid.NewWithString("milos@jackal.im", true)
	o2 := &Occupant{
		Nick: "mynick2",
		FullJID: j2,
		OccupantJID: j2,
		affiliation: "owner",
	}
	occMap := make(map[string]*Occupant)
	occMap[o1.Nick] = o1
	occMap[o2.Nick] = o2
	userMap := make(map[string]*Occupant)
	userMap[o1.FullJID.ToBareJID().String()] = o1
	userMap[o2.FullJID.ToBareJID().String()] = o2

	r := Room{
		RoomJID: rJID,
		Config: &rc,
		NickToOccupant: occMap,
		UserToOccupant: userMap,
	}

	admins := r.GetAdmins()
	owners := r.GetOwners()

	require.NotNil(t, admins)
	require.Equal(t, len(admins), 1)
	require.Equal(t, admins[0], j1.String())

	require.NotNil(t, owners)
	require.Equal(t, len(owners), 1)
	require.Equal(t, owners[0], j2.String())
}
