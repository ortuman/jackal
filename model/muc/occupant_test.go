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

func TestModelOccupant(t *testing.T){
	jOcc, _ := jid.NewWithString("room@conference.jackal.im/mynick", true)
	jFull, _ := jid.NewWithString("ortuman@jackal.im/laptop", true)
	o1 := Occupant{
		OccupantJID: jOcc,
		Nick: "mynick",
		BareJID: jFull,
		affiliation: "owner",
		role: "moderator",
	}

	buf := new(bytes.Buffer)
	require.Nil(t, o1.ToBytes(buf))

	o2 := Occupant{}
	require.Nil(t, o2.FromBytes(buf))

	assert.EqualValues(t, o1, o2)
}

func TestOccupantRoleAndAffiliation(t *testing.T){
	o := Occupant{
		affiliation: "owner",
		role: "moderator",
	}

	require.True(t, o.IsOwner())
	require.False(t, o.IsAdmin())
	require.False(t, o.IsMember())
	require.False(t, o.IsOutcast())

	require.True(t, o.IsModerator())
	require.False(t, o.IsParticipant())
	require.False(t, o.IsVisitor())

	err := o.SetAffiliation("fail")
	require.NotNil(t, err)
	err = o.SetAffiliation(admin)
	require.True(t, o.IsAdmin())

	err = o.SetRole("fail")
	require.NotNil(t, err)
	err = o.SetRole(moderator)
	require.True(t, o.IsModerator())
}
