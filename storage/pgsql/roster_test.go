/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"database/sql/driver"
	"encoding/json"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestInsertRosterItem(t *testing.T) {
	groups := []string{"Buddies", "Family"}
	ri := rostermodel.Item{
		Username:     "user",
		JID:          "contact@jid",
		Name:         "a name",
		Subscription: "both",
		Ask:          false,
		Ver:          1,
		Groups:       groups,
	}

	groupsBytes, _ := json.Marshal(groups)
	args := []driver.Value{
		ri.Username,
		ri.JID,
		ri.Name,
		ri.Subscription,
		groupsBytes,
		ri.Ask,
		ri.Username,
	}

	s, mock := NewMock()

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO roster_versions (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(ri.Username).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO roster_items (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("DELETE FROM roster_groups (.+)").
		WithArgs("user", "contact@jid").
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec("INSERT INTO roster_groups (.+)").
		WithArgs("user", "contact@jid", "Buddies").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO roster_groups (.+)").
		WithArgs("user", "contact@jid", "Family").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs(ri.Username).
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(1, 0))

	_, err := s.InsertOrUpdateRosterItem(&ri)
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestDeleteRosterItem(t *testing.T) {
	s, mock := NewMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO roster_versions (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs("user").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_groups (.+)").
		WithArgs("user", "contact").WillReturnResult(sqlmock.NewResult(0, 1))
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
		WithArgs("user").WillReturnError(errGeneric)
	mock.ExpectRollback()

	_, err = s.DeleteRosterItem("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}

func TestFetchRosterItems(t *testing.T) {
	var riColumns = []string{"user", "contact", "name", "subscription", "`groups`", "ask", "ver"}

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
		WillReturnError(errGeneric)

	_, _, err = s.FetchRosterItems("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnRows(sqlmock.NewRows(riColumns).AddRow("ortuman", "romeo", "Romeo", "both", "", false, 0))

	_, err = s.FetchRosterItem("ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnRows(sqlmock.NewRows(riColumns))

	ri, _ := s.FetchRosterItem("ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, ri)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnError(errGeneric)

	_, err = s.FetchRosterItem("ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}

func TestInsertRosterNotification(t *testing.T) {
	rn := rostermodel.Notification{
		Contact:  "ortuman",
		JID:      "romeo",
		Presence: &xmpp.Presence{},
	}
	presenceXML := rn.Presence.String()

	args := []driver.Value{
		rn.Contact,
		rn.JID,
		presenceXML,
		presenceXML,
	}
	s, mock := NewMock()
	mock.ExpectExec("INSERT INTO roster_notifications (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOrUpdateRosterNotification(&rn)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectExec("INSERT INTO roster_notifications (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(args...).
		WillReturnError(errGeneric)

	err = s.InsertOrUpdateRosterNotification(&rn)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}

func TestDeleteRosterNotification(t *testing.T) {
	s, mock := NewMock()
	mock.ExpectExec("DELETE FROM roster_notifications (.+)").
		WithArgs("user", "contact").WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeleteRosterNotification("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = NewMock()
	mock.ExpectExec("DELETE FROM roster_notifications (.+)").
		WithArgs("user", "contact").WillReturnError(errGeneric)

	err = s.DeleteRosterNotification("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}

func TestFetchRosterNotifications(t *testing.T) {
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
		WillReturnError(errGeneric)

	_, err = s.FetchRosterNotifications("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)

	s, mock = NewMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(rnColumns).AddRow("romeo", "contact", "<priority>8"))

	_, err = s.FetchRosterNotifications("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)
}
