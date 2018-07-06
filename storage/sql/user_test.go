/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
	"github.com/stretchr/testify/require"
)

func TestMySQLStorageInsertUser(t *testing.T) {
	from, _ := jid.NewWithString("ortuman@jackal.im/Psi+", true)
	to, _ := jid.NewWithString("ortuman@jackal.im", true)
	p := xml.NewPresence(from, to, xml.UnavailableType)

	user := model.User{Username: "ortuman", Password: "1234", LastPresence: p}

	s, mock := NewMock()
	mock.ExpectExec("INSERT INTO users (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "1234", p.String(), "1234", p.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOrUpdateUser(&user)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectExec("INSERT INTO users (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "1234", p.String(), "1234", p.String()).
		WillReturnError(errMySQLStorage)
	err = s.InsertOrUpdateUser(&user)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageDeleteUser(t *testing.T) {
	s, mock := NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_items (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_versions (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM private_storage (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM vcards (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM users (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnError(errMySQLStorage)
	mock.ExpectRollback()

	err = s.DeleteUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchUser(t *testing.T) {
	from, _ := jid.NewWithString("ortuman@jackal.im/Psi+", true)
	to, _ := jid.NewWithString("ortuman@jackal.im", true)
	p := xml.NewPresence(from, to, xml.UnavailableType)

	var userColumns = []string{"username", "password", "last_presence", "last_presence_at"}

	s, mock := NewMock()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(userColumns))

	usr, err := s.FetchUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, usr)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow("ortuman", "1234", p.String(), time.Now()))
	_, err = s.FetchUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").WillReturnError(errMySQLStorage)
	_, err = s.FetchUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageUserExists(t *testing.T) {
	countColums := []string{"count"}

	s, mock := NewMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(countColums).AddRow(1))

	ok, err := s.UserExists("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.True(t, ok)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM users (.+)").
		WithArgs("romeo").
		WillReturnError(errMySQLStorage)
	_, err = s.UserExists("romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}
