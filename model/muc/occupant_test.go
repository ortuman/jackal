/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mucmodel

import (
	"bytes"
	"testing"

	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOccupant_Bytes(t *testing.T) {
	jOcc, _ := jid.NewWithString("room@conference.jackal.im/mynick", true)
	jFull, _ := jid.NewWithString("ortuman@jackal.im/laptop", true)
	o1 := Occupant{
		OccupantJID: jOcc,
		BareJID:     jFull.ToBareJID(),
		affiliation: "owner",
		role:        "moderator",
		resources:   make(map[string]bool),
	}
	o1.resources[jFull.Resource()] = true

	buf := new(bytes.Buffer)
	require.Nil(t, o1.ToBytes(buf))

	o2 := Occupant{}
	require.Nil(t, o2.FromBytes(buf))

	assert.EqualValues(t, o1, o2)
}

func TestOccupant_RoleAndAffiliation(t *testing.T) {
	jo, _ := jid.NewWithString("room@conference.jackal.im/owner", true)
	o := &Occupant{
		OccupantJID: jo,
		affiliation: "",
		role:        "visitor",
	}

	require.False(t, o.IsOwner())
	require.False(t, o.IsAdmin())
	require.False(t, o.IsMember())
	require.False(t, o.IsOutcast())
	require.True(t, o.HasNoAffiliation())

	require.True(t, o.IsVisitor())
	require.False(t, o.IsParticipant())
	require.False(t, o.IsModerator())

	err := o.SetAffiliation("fail")
	require.NotNil(t, err)
	err = o.SetAffiliation("owner")
	require.Nil(t, err)

	err = o.SetRole("fail")
	require.NotNil(t, err)
	err = o.SetRole(moderator)
	require.True(t, o.IsModerator())

	jo2, _ := jid.NewWithString("room@conference.jackal.im/admin", true)
	o2 := &Occupant{
		OccupantJID: jo2,
		affiliation: "admin",
		role:        "moderator",
	}

	require.True(t, o.HasHigherAffiliation(o2))
	require.False(t, o2.HasHigherAffiliation(o))
	require.False(t, o.CanChangeRole(o2, "fail"))
	require.True(t, o.CanChangeRole(o2, "visitor"))
	require.True(t, o.CanChangeAffiliation(o2, "owner"))
	require.False(t, o.CanChangeAffiliation(o2, "fail"))
	require.False(t, o2.CanChangeAffiliation(o, "admin"))
}

func TestOccupant_Resources(t *testing.T) {
	jOcc, _ := jid.NewWithString("room@conference.jackal.im/mynick", true)
	jFull, _ := jid.NewWithString("ortuman@jackal.im/laptop", true)

	o, err := NewOccupant(jOcc, jFull)
	require.NotNil(t, err)
	require.Nil(t, o)
	o, err = NewOccupant(jOcc, jFull.ToBareJID())
	require.Nil(t, err)
	require.NotNil(t, o)

	require.False(t, o.HasResource("laptop"))
	o.AddResource("laptop")
	require.True(t, o.HasResource("laptop"))
	require.Len(t, o.GetAllResources(), 1)
	o.DeleteResource("laptop")
	require.False(t, o.HasResource("laptop"))
}
