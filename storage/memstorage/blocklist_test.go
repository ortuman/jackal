/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"testing"

	"github.com/ortuman/jackal/storage/model"
	"github.com/stretchr/testify/require"
)

func TestMockStorageInsertOrUpdateBlockListItems(t *testing.T) {
	items := []model.BlockListItem{
		{"ortuman", "user@jackal.im"},
		{"ortuman", "romeo@jackal.im"},
		{"ortuman", "juliet@jackal.im"},
	}
	s := New()
	s.ActivateMockedError()
	require.Equal(t, ErrMockedError, s.InsertOrUpdateBlockListItems(items))
	s.DeactivateMockedError()

	s.InsertOrUpdateBlockListItems(items)

	s.ActivateMockedError()
	_, err := s.FetchBlockListItems("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DeactivateMockedError()

	sItems, _ := s.FetchBlockListItems("ortuman")
	require.Equal(t, items, sItems)
}

func TestMockStorageDeleteBlockListItems(t *testing.T) {
	items := []model.BlockListItem{
		{"ortuman", "user@jackal.im"},
		{"ortuman", "romeo@jackal.im"},
		{"ortuman", "juliet@jackal.im"},
	}
	s := New()
	s.InsertOrUpdateBlockListItems(items)

	delItems := []model.BlockListItem{{"ortuman", "romeo@jackal.im"}}
	s.ActivateMockedError()
	require.Equal(t, ErrMockedError, s.DeleteBlockListItems(delItems))
	s.DeactivateMockedError()

	s.DeleteBlockListItems(delItems)
	sItems, _ := s.FetchBlockListItems("ortuman")
	require.Equal(t, []model.BlockListItem{
		{"ortuman", "user@jackal.im"},
		{"ortuman", "juliet@jackal.im"},
	}, sItems)
}
