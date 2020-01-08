/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"
	"reflect"
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_BlockListItems(t *testing.T) {
	t.Parallel()

	s, teardown := newBlockListMock()
	defer teardown()

	items := []model.BlockListItem{
		{Username: "ortuman", JID: "juliet@jackal.im"},
		{Username: "ortuman", JID: "user@jackal.im"},
		{Username: "ortuman", JID: "romeo@jackal.im"},
	}

	require.Nil(t, s.InsertBlockListItem(context.Background(), &items[0]))
	require.Nil(t, s.InsertBlockListItem(context.Background(), &items[1]))
	require.Nil(t, s.InsertBlockListItem(context.Background(), &items[2]))

	sItems, err := s.FetchBlockListItems(context.Background(), "ortuman")
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(items, sItems))

	err = s.DeleteBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "user@jackal.im"})
	require.Nil(t, err)

	items = append(items[:1], items[2:]...)

	sItems, err = s.FetchBlockListItems(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, items, sItems)

	require.Nil(t, s.DeleteBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "juliet@jackal.im"}))
	require.Nil(t, s.DeleteBlockListItem(context.Background(), &model.BlockListItem{Username: "ortuman", JID: "romeo@jackal.im"}))

	sItems, _ = s.FetchBlockListItems(context.Background(), "ortuman")
	require.Equal(t, 0, len(sItems))
}

func newBlockListMock() (*badgerDBBlockList, func()) {
	t := newT()
	return &badgerDBBlockList{badgerDBStorage: newStorage(t.db)}, func() {
		t.teardown()
	}
}
