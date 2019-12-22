/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertOrUpdateBlockListItems(t *testing.T) {
	items := []model.BlockListItem{
		{Username: "ortuman", JID: "user@jackal.im"},
		{Username: "ortuman", JID: "romeo@jackal.im"},
		{Username: "ortuman", JID: "juliet@jackal.im"},
	}
	s := New()
	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.InsertBlockListItem(&model.BlockListItem{Username: "ortuman", JID: "user@jackal.im"}))
	s.DisableMockedError()

	require.Nil(t, s.InsertBlockListItem(&model.BlockListItem{Username: "ortuman", JID: "user@jackal.im"}))
	require.Nil(t, s.InsertBlockListItem(&model.BlockListItem{Username: "ortuman", JID: "romeo@jackal.im"}))
	require.Nil(t, s.InsertBlockListItem(&model.BlockListItem{Username: "ortuman", JID: "juliet@jackal.im"}))

	s.EnableMockedError()
	_, err := s.FetchBlockListItems("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()

	sItems, _ := s.FetchBlockListItems("ortuman")
	require.Equal(t, items, sItems)
}

func TestMemoryStorage_DeleteBlockListItems(t *testing.T) {
	s := New()
	require.Nil(t, s.InsertBlockListItem(&model.BlockListItem{Username: "ortuman", JID: "user@jackal.im"}))
	require.Nil(t, s.InsertBlockListItem(&model.BlockListItem{Username: "ortuman", JID: "romeo@jackal.im"}))
	require.Nil(t, s.InsertBlockListItem(&model.BlockListItem{Username: "ortuman", JID: "juliet@jackal.im"}))

	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.DeleteBlockListItem(&model.BlockListItem{Username: "ortuman", JID: "romeo@jackal.im"}))
	s.DisableMockedError()

	require.Nil(t, s.DeleteBlockListItem(&model.BlockListItem{Username: "ortuman", JID: "romeo@jackal.im"}))

	sItems, _ := s.FetchBlockListItems("ortuman")
	require.Equal(t, []model.BlockListItem{
		{Username: "ortuman", JID: "user@jackal.im"},
		{Username: "ortuman", JID: "juliet@jackal.im"},
	}, sItems)
}
