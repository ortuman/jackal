/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	rostermodel "github.com/ortuman/jackal/model/roster"
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

	s, mock := newRosterMock()

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

	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs(ri.Username).
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(1, 0))

	mock.ExpectCommit()

	_, err := s.UpsertRosterItem(context.Background(), &ri)
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestDeleteRosterItem(t *testing.T) {
	s, mock := newRosterMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO roster_versions (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs("user").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_groups (.+)").
		WithArgs("user", "contact").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_items (.+)").
		WithArgs("user", "contact").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(1, 0))
	mock.ExpectCommit()

	_, err := s.DeleteRosterItem(context.Background(), "user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newRosterMock()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO roster_versions (.+)").
		WithArgs("user").WillReturnError(errGeneric)
	mock.ExpectRollback()

	_, err = s.DeleteRosterItem(context.Background(), "user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}

func TestFetchRosterItems(t *testing.T) {
	var riColumns = []string{"user", "contact", "name", "subscription", "`groups`", "ask", "ver"}

	s, mock := newRosterMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(riColumns).AddRow("ortuman", "romeo", "Romeo", "both", "", false, 0))
	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(0, 0))

	rosterItems, _, err := s.FetchRosterItems(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 1, len(rosterItems))

	s, mock = newRosterMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman").
		WillReturnError(errGeneric)

	_, _, err = s.FetchRosterItems(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)

	s, mock = newRosterMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnRows(sqlmock.NewRows(riColumns).AddRow("ortuman", "romeo", "Romeo", "both", "", false, 0))

	_, err = s.FetchRosterItem(context.Background(), "ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newRosterMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnRows(sqlmock.NewRows(riColumns))

	ri, _ := s.FetchRosterItem(context.Background(), "ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, ri)

	s, mock = newRosterMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnError(errGeneric)

	_, err = s.FetchRosterItem(context.Background(), "ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)

	var riColumns2 = []string{"ris.user", "ris.contact", "ris.name", "ris.subscription", "ris.groups", "ris.ask", "ris.ver"}
	s, mock = newRosterMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_items ris LEFT JOIN roster_groups g ON ris.username = g.username (.+)").
		WithArgs("ortuman", "Family").
		WillReturnRows(sqlmock.NewRows(riColumns2).AddRow("ortuman", "romeo", "Romeo", "both", `["Family"]`, false, 0))
	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(0, 0))

	_, _, err = s.FetchRosterItemsInGroups(context.Background(), "ortuman", []string{"Family"})
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
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
	s, mock := newRosterMock()
	mock.ExpectExec("INSERT INTO roster_notifications (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.UpsertRosterNotification(context.Background(), &rn)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newRosterMock()
	mock.ExpectExec("INSERT INTO roster_notifications (.+) ON CONFLICT (.+) DO UPDATE SET (.+)").
		WithArgs(args...).
		WillReturnError(errGeneric)

	err = s.UpsertRosterNotification(context.Background(), &rn)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}

func TestDeleteRosterNotification(t *testing.T) {
	s, mock := newRosterMock()
	mock.ExpectExec("DELETE FROM roster_notifications (.+)").
		WithArgs("user", "contact").WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeleteRosterNotification(context.Background(), "user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newRosterMock()
	mock.ExpectExec("DELETE FROM roster_notifications (.+)").
		WithArgs("user", "contact").WillReturnError(errGeneric)

	err = s.DeleteRosterNotification(context.Background(), "user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)
}

func TestFetchRosterNotifications(t *testing.T) {
	var rnColumns = []string{"user", "contact", "elements"}

	s, mock := newRosterMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(rnColumns).AddRow("romeo", "contact", "<priority>8</priority>"))

	rosterNotifications, err := s.FetchRosterNotifications(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 1, len(rosterNotifications))

	s, mock = newRosterMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(rnColumns))

	rosterNotifications, err = s.FetchRosterNotifications(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 0, len(rosterNotifications))

	s, mock = newRosterMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnError(errGeneric)

	_, err = s.FetchRosterNotifications(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errGeneric, err)

	s, mock = newRosterMock()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(rnColumns).AddRow("romeo", "contact", "<priority>8"))

	_, err = s.FetchRosterNotifications(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)
}

func TestStorageFetchRosterGroups(t *testing.T) {
	s, mock := newRosterMock()
	mock.ExpectQuery("SELECT `group` FROM roster_groups WHERE username = (.+) GROUP BY (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows([]string{"group"}).
			AddRow("Contacts").
			AddRow("News"))

	groups, err := s.FetchRosterGroups(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	require.Equal(t, 2, len(groups))
	require.Equal(t, "Contacts", groups[0])
	require.Equal(t, "News", groups[1])

	s, mock = newRosterMock()
	mock.ExpectQuery("SELECT `group` FROM roster_groups WHERE username = (.+) GROUP BY (.+)").
		WithArgs("ortuman").
		WillReturnError(errGeneric)

	groups, err = s.FetchRosterGroups(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())

	require.Nil(t, groups)
	require.NotNil(t, err)
	require.Equal(t, errGeneric, err)
}

func newRosterMock() (*pgSQLRoster, sqlmock.Sqlmock) {
	s, sqlMock := newStorageMock()
	return &pgSQLRoster{
		pgSQLStorage: s,
	}, sqlMock
}
