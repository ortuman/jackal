/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ortuman/jackal/xmpp/jid"
)

func TestModelRoom(t *testing.T){
	rJID, _ := jid.NewWithString("ortuman@jackal.im", true)
	rc := RoomConfig{
		Public: true,
		Persistent: true,
		PwdProtected: true,
		Password: "pwd",
		Open: true,
		Moderated: true,
		NonAnonymous: false,
	}

	jOcc, _ := jid.NewWithString("room@conference.jackal.im/mynick", true)
	jFull, _ := jid.NewWithString("ortuman@jackal.im/laptop", true)
	o := &Occupant{
		OccupantJID: jOcc,
		Nick: "mynick",
		FullJID: jFull,
		Affiliation: "owner",
		Role: "moderator",
	}
	occMap := make(map[string]*Occupant)
	occMap[o.Nick] = o

	r1 := Room{
		Name: "Test Room",
		RoomJID: rJID,
		Desc: "Test Description",
		Config: &rc,
		OccupantsCnt: 1,
		Occupants: occMap,
		Locked: true,
	}

	buf := new(bytes.Buffer)
	require.Nil(t, r1.ToBytes(buf))

	r2 := Room{}
	require.Nil(t, r2.FromBytes(buf))
	require.Equal(t, r1.Name, r2.Name)
	require.Equal(t, r1.RoomJID.String(), r2.RoomJID.String())
	require.Equal(t, r1.Desc, r2.Desc)
	require.Equal(t, rc.Password, r2.Config.Password)
	require.Equal(t, r1.OccupantsCnt, r2.OccupantsCnt)
	require.Equal(t, o.FullJID, r2.Occupants[o.Nick].FullJID)
	require.Equal(t, r1.Locked, r2.Locked)
}
