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

func TestMemoryStorage_InsertRoom(t *testing.T) {
	r := GetTestRoom()
	s := NewRoom()
	EnableMockedError()
	err := s.UpsertRoom(context.Background(), r)
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	err = s.UpsertRoom(context.Background(), r)
	require.Nil(t, err)
}

func TestMemoryStorage_RoomExists(t *testing.T) {
	j, _ := jid.NewWithString("testroom@conference.jackal.im", true)
	s := NewRoom()
	EnableMockedError()
	_, err := s.RoomExists(context.Background(), j)
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	ok, err := s.RoomExists(context.Background(), j)
	require.Nil(t, err)
	require.False(t, ok)

	r := GetTestRoom()
	require.Equal(t, r.RoomJID, j)
	s.saveEntity(roomKey(r.RoomJID), r)
	ok, err = s.RoomExists(context.Background(), j)
	require.Nil(t, err)
	require.True(t, ok)
}

func TestMemoryStorage_FetchRoom(t *testing.T) {
	j, _ := jid.NewWithString("testroom@conference.jackal.im", true)
	r := GetTestRoom()
	s := NewRoom()
	_ = s.UpsertRoom(context.Background(), r)

	EnableMockedError()
	_, err := s.FetchRoom(context.Background(), j)
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	notInMemoryJID, _ := jid.NewWithString("faketestroom@conference.jackal.im", true)
	roomFromMemory, _ := s.FetchRoom(context.Background(), notInMemoryJID)
	require.Nil(t, roomFromMemory)

	roomFromMemory, _ = s.FetchRoom(context.Background(), j)
	require.NotNil(t, roomFromMemory)
	assert.EqualValues(t, r, roomFromMemory)
}

func TestMemoryStorage_DeleteRoom(t *testing.T) {
	j, _ := jid.NewWithString("testroom@conference.jackal.im", true)
	r := GetTestRoom()
	s := NewRoom()
	_ = s.UpsertRoom(context.Background(), r)

	EnableMockedError()
	require.Equal(t, ErrMocked, s.DeleteRoom(context.Background(), j))
	DisableMockedError()
	require.Nil(t, s.DeleteRoom(context.Background(), j))

	room, _ := s.FetchRoom(context.Background(), j)
	require.Nil(t, room)
}

func GetTestRoom() *mucmodel.Room {
	rc := mucmodel.RoomConfig{
		Public:       true,
		Persistent:   true,
		PwdProtected: false,
		Open:         true,
		Moderated:    false,
	}
	j, _ := jid.NewWithString("testroom@conference.jackal.im", true)

	return &mucmodel.Room{
		Name:           "testRoom",
		RoomJID:        j,
		Desc:           "Room for Testing",
		Config:         &rc,
		OccupantsCnt:   0,
		NickToOccupant: make(map[string]*mucmodel.Occupant),
		UserToOccupant: make(map[string]*mucmodel.Occupant),
		Locked:         false,
	}
}
