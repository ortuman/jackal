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

func TestMockStorageInsertOrUpdateBlockListItems(t *testing.T) {
	items := []model.BlockListItem{
		{Username: "ortuman", JID: "user@jackal.im"},
		{Username: "ortuman", JID: "romeo@jackal.im"},
		{Username: "ortuman", JID: "juliet@jackal.im"},
	}
	s := New()
	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.InsertBlockListItems(items))
	s.DisableMockedError()

	s.InsertBlockListItems(items)

	s.EnableMockedError()
	_, err := s.FetchBlockListItems("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()

	sItems, _ := s.FetchBlockListItems("ortuman")
	require.Equal(t, items, sItems)
}

func TestMockStorageDeleteBlockListItems(t *testing.T) {
	items := []model.BlockListItem{
		{Username: "ortuman", JID: "user@jackal.im"},
		{Username: "ortuman", JID: "romeo@jackal.im"},
		{Username: "ortuman", JID: "juliet@jackal.im"},
	}
	s := New()
	s.InsertBlockListItems(items)

	delItems := []model.BlockListItem{{Username: "ortuman", JID: "romeo@jackal.im"}}
	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.DeleteBlockListItems(delItems))
	s.DisableMockedError()

	s.DeleteBlockListItems(delItems)
	sItems, _ := s.FetchBlockListItems("ortuman")
	require.Equal(t, []model.BlockListItem{
		{Username: "ortuman", JID: "user@jackal.im"},
		{Username: "ortuman", JID: "juliet@jackal.im"},
	}, sItems)
}
