/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"sort"
	"testing"

	"github.com/ortuman/jackal/storage/model"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_BlockListItems(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	items := []model.BlockListItem{
		{"ortuman", "juliet@jackal.im"},
		{"ortuman", "user@jackal.im"},
		{"ortuman", "romeo@jackal.im"},
	}
	sort.Slice(items, func(i, j int) bool { return items[i].JID < items[j].JID })

	err := h.db.InsertBlockListItems(items)
	require.Nil(t, err)

	sItems, err := h.db.FetchBlockListItems("ortuman")
	sort.Slice(sItems, func(i, j int) bool { return sItems[i].JID < sItems[j].JID })
	require.Nil(t, err)
	require.Equal(t, items, sItems)

	items = append(items[:1], items[2:]...)
	h.db.DeleteBlockListItems([]model.BlockListItem{{"ortuman", "romeo@jackal.im"}})

	sItems, err = h.db.FetchBlockListItems("ortuman")
	sort.Slice(items, func(i, j int) bool { return items[i].JID < items[j].JID })
	require.Nil(t, err)
	require.Equal(t, items, sItems)

	err = h.db.DeleteBlockListItems(items)
	require.Nil(t, err)
	sItems, _ = h.db.FetchBlockListItems("ortuman")
	require.Equal(t, 0, len(sItems))
}
