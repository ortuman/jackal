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

func TestModelOccupant(t *testing.T){
	jOcc, _ := jid.NewWithString("room@conference.jackal.im/mynick", true)
	jFull, _ := jid.NewWithString("ortuman@jackal.im/laptop", true)
	o1 := Occupant{
		OccupantJID: jOcc,
		Nick: "mynick",
		FullJID: jFull,
		Affiliation: "owner",
		Role: "moderator",
	}

	buf := new(bytes.Buffer)
	require.Nil(t, o1.ToBytes(buf))

	o2 := Occupant{}
	require.Nil(t, o2.FromBytes(buf))
	require.Equal(t, o1.OccupantJID.String(), o2.OccupantJID.String())
	require.Equal(t, o1.Nick, o2.Nick)
	require.Equal(t, o1.FullJID.String(), o2.FullJID.String())
	require.Equal(t, o1.Affiliation, o2.Affiliation)
	require.Equal(t, o1.Role, o2.Role)
}
