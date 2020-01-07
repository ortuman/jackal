/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"
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
	s := NewBlockList()
	EnableMockedError()
	require.Equal(t, ErrMocked, s.InsertBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "user@jackal.im"}))
	DisableMockedError()

	require.Nil(t, s.InsertBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "user@jackal.im"}))
	require.Nil(t, s.InsertBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "romeo@jackal.im"}))
	require.Nil(t, s.InsertBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "juliet@jackal.im"}))

	EnableMockedError()
	_, err := s.FetchBlockListItems(context.Background(), "ortuman")
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	sItems, _ := s.FetchBlockListItems(context.Background(), "ortuman")
	require.Equal(t, items, sItems)
}

func TestMemoryStorage_DeleteBlockListItems(t *testing.T) {
	s := NewBlockList()
	require.Nil(t, s.InsertBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "user@jackal.im"}))
	require.Nil(t, s.InsertBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "romeo@jackal.im"}))
	require.Nil(t, s.InsertBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "juliet@jackal.im"}))

	EnableMockedError()
	require.Equal(t, ErrMocked, s.DeleteBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "romeo@jackal.im"}))
	DisableMockedError()

	require.Nil(t, s.DeleteBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "romeo@jackal.im"}))

	sItems, _ := s.FetchBlockListItems(context.Background(), "ortuman")
	require.Equal(t, []model.BlockListItem{
		{Username: "ortuman", JID: "user@jackal.im"},
		{Username: "ortuman", JID: "juliet@jackal.im"},
	}, sItems)
}
