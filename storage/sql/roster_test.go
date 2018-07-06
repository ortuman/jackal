/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

func TestMySQLStorageInsertRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := rostermodel.Item{"user", "contact", "a name", "both", false, 1, g}

	args := []driver.Value{
		ri.Username,
		ri.JID,
		ri.Name,
		ri.Subscription,
		"general;friends",
		ri.Ask,
		ri.Username,
		ri.Name,
		ri.Subscription,
		"general;friends",
		ri.Ask,
	}

	s, mock := NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO roster_versions (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("user").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO roster_items (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(1, 0))

	_, err := s.InsertOrUpdateRosterItem(&ri)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestMySQLStorageDeleteRosterItem(t *testing.T) {
	s, mock := NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO roster_versions (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("user").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_items (.+)").
		WithArgs("user", "contact").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(1, 0))

	_, err := s.DeleteRosterItem("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO roster_versions (.+)").
		WithArgs("user").WillReturnError(errMySQLStorage)
	mock.ExpectRollback()

	_, err = s.DeleteRosterItem("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchRosterItems(t *testing.T) {
	var riColumns = []string{"user", "contact", "name", "subscription", "groups", "ask", "ver"}

	s, mock := NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(riColumns).AddRow("ortuman", "romeo", "Romeo", "both", "", false, 0))
	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(0, 0))

	rosterItems, _, err := s.FetchRosterItems("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 1, len(rosterItems))

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	_, _, err = s.FetchRosterItems("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnRows(sqlmock.NewRows(riColumns).AddRow("ortuman", "romeo", "Romeo", "both", "", false, 0))

	ri, err := s.FetchRosterItem("ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnRows(sqlmock.NewRows(riColumns))

	ri, err = s.FetchRosterItem("ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, ri)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnError(errMySQLStorage)

	_, err = s.FetchRosterItem("ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageInsertRosterNotification(t *testing.T) {
	rn := rostermodel.Notification{
		"ortuman",
		"romeo",
		&xml.Presence{},
	}
	presenceXML := rn.Presence.String()

	args := []driver.Value{
		rn.Contact,
		rn.JID,
		presenceXML,
		presenceXML,
	}
	s, mock := NewMock()
	mock.ExpectExec("INSERT INTO roster_notifications (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOrUpdateRosterNotification(&rn)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectExec("INSERT INTO roster_notifications (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(args...).
		WillReturnError(errMySQLStorage)

	err = s.InsertOrUpdateRosterNotification(&rn)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageDeleteRosterNotification(t *testing.T) {
	s, mock := NewMock()
	mock.ExpectExec("DELETE FROM roster_notifications (.+)").
		WithArgs("user", "contact").WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeleteRosterNotification("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectExec("DELETE FROM roster_notifications (.+)").
		WithArgs("user", "contact").WillReturnError(errMySQLStorage)

	err = s.DeleteRosterNotification("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchRosterNotifications(t *testing.T) {
	var rnColumns = []string{"user", "contact", "elements"}

	s, mock := NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(rnColumns).AddRow("romeo", "contact", "<priority>8</priority>"))

	rosterNotifications, err := s.FetchRosterNotifications("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 1, len(rosterNotifications))

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(rnColumns))

	rosterNotifications, err = s.FetchRosterNotifications("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 0, len(rosterNotifications))

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	_, err = s.FetchRosterNotifications("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(rnColumns).AddRow("romeo", "contact", "<priority>8"))

	_, err = s.FetchRosterNotifications("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)
}
