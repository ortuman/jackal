/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"
	"testing"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertOccupant(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	o := mucmodel.Occupant{
		OccupantJID: j,
		BareJID:     j,
	}
	o.SetAffiliation("owner")
	o.SetRole("moderator")
	s := NewOccupant()
	EnableMockedError()
	err := s.UpsertOccupant(context.Background(), &o)
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	err = s.UpsertOccupant(context.Background(), &o)
	require.Nil(t, err)
}

func TestMemoryStorage_OccupantExists(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	s := NewOccupant()
	EnableMockedError()
	_, err := s.OccupantExists(context.Background(), j)
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	ok, err := s.OccupantExists(context.Background(), j)
	require.Nil(t, err)
	require.False(t, ok)

	o := mucmodel.Occupant{
		OccupantJID: j,
		BareJID:     j,
	}
	o.SetAffiliation("owner")
	o.SetRole("moderator")
	s.saveEntity(occKey(j), &o)
	ok, err = s.OccupantExists(context.Background(), j)
	require.Nil(t, err)
	require.True(t, ok)
}

func TestMemoryStorage_FetchOccupant(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	o := &mucmodel.Occupant{
		OccupantJID: j,
		BareJID:     j,
		Resources:   make(map[string]bool),
	}
	o.SetAffiliation("owner")
	o.SetRole("moderator")
	s := NewOccupant()
	_ = s.UpsertOccupant(context.Background(), o)

	EnableMockedError()
	_, err := s.FetchOccupant(context.Background(), j)
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	notInMemoryJID, _ := jid.NewWithString("romeo@jackal.im/yard", true)
	occ, _ := s.FetchOccupant(context.Background(), notInMemoryJID)
	require.Nil(t, occ)

	occ, _ = s.FetchOccupant(context.Background(), j)
	require.NotNil(t, occ)
	assert.EqualValues(t, o, occ)
}

func TestMemoryStorage_DeleteOccupant(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	o := mucmodel.Occupant{
		OccupantJID: j,
		BareJID:     j,
	}
	o.SetAffiliation("owner")
	o.SetRole("moderator")
	s := NewOccupant()
	_ = s.UpsertOccupant(context.Background(), &o)

	EnableMockedError()
	require.Equal(t, ErrMocked, s.DeleteOccupant(context.Background(), j))
	DisableMockedError()
	require.Nil(t, s.DeleteOccupant(context.Background(), j))

	occ, _ := s.FetchOccupant(context.Background(), j)
	require.Nil(t, occ)
}
