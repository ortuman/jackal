/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"
	"testing"

	mucmodel "github.com/ortuman/jackal/model/muc"
	"github.com/stretchr/testify/require"
	"github.com/ortuman/jackal/xmpp/jid"
)

func TestMemoryStorage_InsertOccupant(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	o := mucmodel.Occupant{
		OccupantJID: j,
		Nick: "ortuman",
		FullJID: j,
		Affiliation: "Owner",
		Role: "Moderator"}
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
		Nick: "ortuman",
		FullJID: j,
		Affiliation: "Owner",
		Role: "Moderator"}
	s.saveEntity(occKey(j), &o)
	ok, err = s.OccupantExists(context.Background(), j)
	require.Nil(t, err)
	require.True(t, ok)
}

func TestMemoryStorage_FetchOccupant(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	o := mucmodel.Occupant{
		OccupantJID: j,
		Nick: "ortuman",
		FullJID: j,
		Affiliation: "Owner",
		Role: "Moderator"}
	s := NewOccupant()
	_ = s.UpsertOccupant(context.Background(), &o)

	EnableMockedError()
	_, err := s.FetchOccupant(context.Background(), j)
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	notInMemoryJID, _ := jid.NewWithString("romeo@jackal.im/yard", true)
	occ, _ := s.FetchOccupant(context.Background(), notInMemoryJID)
	require.Nil(t, occ)

	occ, _ = s.FetchOccupant(context.Background(), j)
	require.NotNil(t, occ)
}

func TestMemoryStorage_DeleteOccupant(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	o := mucmodel.Occupant{
		OccupantJID: j,
		Nick: "ortuman",
		FullJID: j,
		Affiliation: "Owner",
		Role: "Moderator"}
	s := NewOccupant()
	_ = s.UpsertOccupant(context.Background(), &o)

	EnableMockedError()
	require.Equal(t, ErrMocked, s.DeleteOccupant(context.Background(), j))
	DisableMockedError()
	require.Nil(t, s.DeleteOccupant(context.Background(), j))

	occ, _ := s.FetchOccupant(context.Background(), j)
	require.Nil(t, occ)
}
