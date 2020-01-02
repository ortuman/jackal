/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/pool"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestMySQLStorageInsertUser(t *testing.T) {
	from, _ := jid.NewWithString("ortuman@jackal.im/Psi+", true)
	to, _ := jid.NewWithString("ortuman@jackal.im", true)
	p := xmpp.NewPresence(from, to, xmpp.UnavailableType)

	user := model.User{Username: "ortuman", Password: "1234", LastPresence: p}

	s, mock := newMock()
	mock.ExpectExec("INSERT INTO users (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "1234", p.String(), "1234", p.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.UpsertUser(context.Background(), &user)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMock()
	mock.ExpectExec("INSERT INTO users (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "1234", p.String(), "1234", p.String()).
		WillReturnError(errMocked)

	err = s.UpsertUser(context.Background(), &user)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMocked, err)
}

func TestMySQLStorageDeleteUser(t *testing.T) {
	s, mock := newMock()
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

	err := s.DeleteUser(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMock()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnError(errMocked)
	mock.ExpectRollback()

	err = s.DeleteUser(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMocked, err)
}

func TestMySQLStorageFetchUser(t *testing.T) {
	from, _ := jid.NewWithString("ortuman@jackal.im/Psi+", true)
	to, _ := jid.NewWithString("ortuman@jackal.im", true)
	p := xmpp.NewPresence(from, to, xmpp.UnavailableType)

	var userColumns = []string{"username", "password", "last_presence", "last_presence_at"}

	s, mock := newMock()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(userColumns))

	usr, _ := s.FetchUser(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, usr)

	s, mock = newMock()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow("ortuman", "1234", p.String(), time.Now()))
	_, err := s.FetchUser(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMock()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").WillReturnError(errMocked)
	_, err = s.FetchUser(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMocked, err)
}

func TestMySQLStorageUserExists(t *testing.T) {
	countCols := []string{"count"}

	s, mock := newMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(countCols).AddRow(1))

	ok, err := s.UserExists(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.True(t, ok)

	s, mock = newMock()
	mock.ExpectQuery("SELECT COUNT(.+) FROM users (.+)").
		WithArgs("romeo").
		WillReturnError(errMocked)
	_, err = s.UserExists(context.Background(), "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMocked, err)
}

func newMock() (*mySQLUser, sqlmock.Sqlmock) {
	s, sqlMock := newStorageMock()
	return &mySQLUser{
		mySQLStorage: s,
		pool:         pool.NewBufferPool(),
	}, sqlMock
}
