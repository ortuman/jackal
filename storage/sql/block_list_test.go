/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/model"
	"github.com/stretchr/testify/require"
)

func TestMySQLStorageInsertBlockListItems(t *testing.T) {
	s, mock := NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO blocklist_items (.+)").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.InsertBlockListItems([]model.BlockListItem{{"ortuman", "noelia@jackal.im"}})
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO blocklist_items (.+)").WillReturnError(errMySQLStorage)
	mock.ExpectRollback()

	err = s.InsertBlockListItems([]model.BlockListItem{{"ortuman", "noelia@jackal.im"}})
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLFetchBlockListItems(t *testing.T) {
	var blockListColumns = []string{"username", "jid"}
	s, mock := NewMock()
	mock.ExpectQuery("SELECT (.+) FROM blocklist_items (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(blockListColumns).AddRow("ortuman", "noelia@jackal.im"))

	_, err := s.FetchBlockListItems("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM blocklist_items (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	_, err = s.FetchBlockListItems("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageDeleteBlockListItems(t *testing.T) {
	s, mock := NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM blocklist_items (.+)").
		WithArgs("ortuman").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	s, mock = NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM blocklist_items (.+)").
		WithArgs("ortuman", "noelia@jackal.im").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	delItems := []model.BlockListItem{{"ortuman", "noelia@jackal.im"}}
	err := s.DeleteBlockListItems(delItems)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM blocklist_items (.+)").
		WillReturnError(errMySQLStorage)
	mock.ExpectRollback()

	err = s.DeleteBlockListItems(delItems)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}
