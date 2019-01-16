/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

const (
	blockListInsert = "INSERT INTO blocklist_items (.+)"
	blockListDelete = "DELETE FROM blocklist_items (.+)"
	blockListSelect = "SELECT (.+) FROM blocklist_items (.+)"
)

// Insert a valid block list item
func TestInsertValidBlockListItem(t *testing.T) {
	s, mock := NewMock()
	items := []model.BlockListItem{{Username: "ortuman", JID: "noelia@jackal.im"}}

	mock.ExpectBegin()
	mock.ExpectExec(blockListInsert).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.InsertBlockListItems(items)
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

// Insert the same row twice to test for key uniqueness validation
func TestInsertDoubleBlockListItem(t *testing.T) {
	s, mock := NewMock()
	items := []model.BlockListItem{{Username: "ortuman", JID: "noelia@jackal.im"}}

	// First insertion will be successful
	mock.ExpectBegin()
	mock.ExpectExec(blockListInsert).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// Second insertion will fail
	mock.ExpectBegin()
	mock.ExpectExec(blockListInsert).WillReturnError(errGeneric)
	mock.ExpectRollback()

	err := s.InsertBlockListItems(items)
	require.Nil(t, err)

	err = s.InsertBlockListItems(items)
	require.Equal(t, errGeneric, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

// Test fetching block list items
func TestFetchBlockListItems(t *testing.T) {
	var blockListColumns = []string{"username", "jid"}
	s, mock := NewMock()

	mock.ExpectQuery(blockListSelect).WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(blockListColumns).AddRow("ortuman", "noelia@jackal.im"))

	_, err := s.FetchBlockListItems("ortuman")
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

// Test error handling on fetching block list items
func TestFetchBlockListItemsError(t *testing.T) {
	s, mock := NewMock()

	mock.ExpectQuery(blockListSelect).
		WithArgs("ortuman").
		WillReturnError(errGeneric)

	_, err := s.FetchBlockListItems("ortuman")
	require.Equal(t, errGeneric, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

// Test deleting an item from the block list
func TestDeleteBlockListItems(t *testing.T) {
	s, mock := NewMock()
	item := model.BlockListItem{Username: "ortuman", JID: "noelia@jackal.im"}

	mock.ExpectBegin()
	mock.ExpectExec(blockListDelete).
		WithArgs(item.Username, item.JID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteBlockListItems([]model.BlockListItem{item})
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

// Test error handling on deleting a row from the block list
func TestDeleteBlockListItemsError(t *testing.T) {
	s, mock := NewMock()
	items := []model.BlockListItem{{Username: "ortuman", JID: "noelia@jackal.im"}}

	mock.ExpectBegin()
	mock.ExpectExec(blockListDelete).WillReturnError(errGeneric)
	mock.ExpectRollback()

	err := s.DeleteBlockListItems(items)
	require.Equal(t, errGeneric, err)
	require.Nil(t, mock.ExpectationsWereMet())
}
