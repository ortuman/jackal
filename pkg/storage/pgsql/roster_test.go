// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pgsqlrepository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	"github.com/lib/pq"
	rostermodel "github.com/ortuman/jackal/pkg/model/roster"
	xmpputil "github.com/ortuman/jackal/pkg/util/xmpp"
	"github.com/stretchr/testify/require"
)

func TestPgSQLRoster_TouchRosterVersion(t *testing.T) {
	// given
	s, mock := newRosterMock()
	mock.ExpectQuery(`INSERT INTO roster_versions \(username\) VALUES \(\$1\) ON CONFLICT \(username\) DO UPDATE SET ver = roster_versions\.ver \+ 1 RETURNING ver`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows([]string{"ver"}).AddRow(1),
		)

	// when
	v, err := s.TouchRosterVersion(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Equal(t, 1, v)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_FetchRosterVersion(t *testing.T) {
	// given
	s, mock := newRosterMock()
	mock.ExpectQuery(`SELECT ver FROM roster_versions WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows([]string{"ver"}).AddRow(1),
		)

	// when
	v, err := s.FetchRosterVersion(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Equal(t, 1, v)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_UpsertRosterItem(t *testing.T) {
	// given
	s, mock := newRosterMock()
	mock.ExpectExec(`INSERT INTO roster_items \(username,jid,name,subscription,groups,ask\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\) ON CONFLICT \(username, jid\) DO UPDATE SET name = \$3, subscription = \$4, groups = \$5, ask = \$6`).
		WithArgs("ortuman", "noelia@jackal.im", "Noelia", "both", `{"VIP","Buddies"}`, true).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		Jid:          "noelia@jackal.im",
		Name:         "Noelia",
		Subscription: "both",
		Groups:       []string{"VIP", "Buddies"},
		Ask:          true,
	})

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_DeleteRosterItem(t *testing.T) {
	// given
	s, mock := newRosterMock()
	mock.ExpectExec(`DELETE FROM roster_items WHERE \(username = \$1 AND jid = \$2\)`).
		WithArgs("ortuman", "noelia@jackal.im").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.DeleteRosterItem(context.Background(), "ortuman", "noelia@jackal.im")

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_DeleteRosterItems(t *testing.T) {
	// given
	s, mock := newRosterMock()
	mock.ExpectExec(`DELETE FROM roster_items WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.DeleteRosterItems(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_FetchRosterItems(t *testing.T) {
	// given
	cols := []string{
		"username",
		"jid",
		"name",
		"subscription",
		"groups",
		"ask",
	}
	s, mock := newRosterMock()
	mock.ExpectQuery(`SELECT username, jid, name, subscription, groups, ask FROM roster_items WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(
				"ortuman",
				"noelia@jackal.im",
				"noelia",
				"both",
				pq.Array([]string{"VIP", "Buddies"}),
				false,
			),
		)

	// when
	ris, err := s.FetchRosterItems(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Len(t, ris, 1)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_FetchItemsInGroups(t *testing.T) {
	// given
	cols := []string{
		"username",
		"jid",
		"name",
		"subscription",
		"groups",
		"ask",
	}
	s, mock := newRosterMock()
	mock.ExpectQuery(`SELECT username, jid, name, subscription, groups, ask FROM roster_items WHERE username = \$1 AND groups @> \$2`).
		WithArgs("ortuman", `{"VIP","Buddies"}`).
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(
				"ortuman",
				"noelia@jackal.im",
				"noelia",
				"both",
				pq.Array([]string{"VIP", "Buddies"}),
				false,
			),
		)

	// when
	ris, err := s.FetchRosterItemsInGroups(context.Background(), "ortuman", []string{"VIP", "Buddies"})

	// then
	require.Nil(t, err)
	require.Len(t, ris, 1)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_FetchRosterItem(t *testing.T) {
	// given
	cols := []string{
		"username",
		"jid",
		"name",
		"subscription",
		"groups",
		"ask",
	}
	s, mock := newRosterMock()
	mock.ExpectQuery(`SELECT username, jid, name, subscription, groups, ask FROM roster_items WHERE \(username = \$1 AND jid = \$2\)`).
		WithArgs("ortuman", "noelia@jackal.im").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(
				"ortuman",
				"noelia@jackal.im",
				"noelia",
				"both",
				pq.Array([]string{"VIP", "Buddies"}),
				false,
			),
		)

	// when
	ri, err := s.FetchRosterItem(context.Background(), "ortuman", "noelia@jackal.im")

	// then
	require.Nil(t, err)
	require.NotNil(t, ri)
	require.Equal(t, "noelia@jackal.im", ri.Jid)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_UpsertRosterNotification(t *testing.T) {
	// given
	fromJID, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	toJID, _ := jid.NewWithString("noelia@jackal.im", true)

	pr := xmpputil.MakePresence(fromJID, toJID, stravaganza.ProbeType, nil)

	prBytes, _ := pr.MarshalBinary()

	s, mock := newRosterMock()
	mock.ExpectExec(`INSERT INTO roster_notifications \(contact,jid,presence\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \(contact, jid\) DO UPDATE SET presence = \$3`).
		WithArgs("ortuman", "noelia@jackal.im", prBytes).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.UpsertRosterNotification(context.Background(), &rostermodel.Notification{
		Contact:  "ortuman",
		Jid:      "noelia@jackal.im",
		Presence: pr.Proto(),
	})

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_DeleteRosterNotification(t *testing.T) {
	// given
	s, mock := newRosterMock()
	mock.ExpectExec(`DELETE FROM roster_notifications WHERE \(contact = \$1 AND jid = \$2\)`).
		WithArgs("ortuman", "noelia@jackal.im").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.DeleteRosterNotification(context.Background(), "ortuman", "noelia@jackal.im")

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_DeleteRosterNotifications(t *testing.T) {
	// given
	s, mock := newRosterMock()
	mock.ExpectExec(`DELETE FROM roster_notifications WHERE contact = \$1`).
		WithArgs("ortuman").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.DeleteRosterNotifications(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_FetchRosterNotification(t *testing.T) {
	// given
	cols := []string{
		"contact",
		"jid",
		"presence",
	}
	fromJID, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	toJID, _ := jid.NewWithString("noelia@jackal.im", true)

	pr := xmpputil.MakePresence(fromJID, toJID, stravaganza.ProbeType, nil)

	prBytes, _ := pr.MarshalBinary()

	s, mock := newRosterMock()
	mock.ExpectQuery(`SELECT contact, jid, presence FROM roster_notifications WHERE \(contact = \$1 AND jid = \$2\)`).
		WithArgs("ortuman", "noelia@jackal.im").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(
				"ortuman",
				"noelia@jackal.im",
				prBytes,
			),
		)

	// when
	rn, err := s.FetchRosterNotification(context.Background(), "ortuman", "noelia@jackal.im")

	// then
	require.Nil(t, err)
	require.NotNil(t, rn)
	require.Equal(t, "noelia@jackal.im", rn.Jid)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_FetchRosterNotifications(t *testing.T) {
	// given
	cols := []string{
		"contact",
		"jid",
		"presence",
	}
	fromJID, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	toJID, _ := jid.NewWithString("noelia@jackal.im", true)

	pr := xmpputil.MakePresence(fromJID, toJID, stravaganza.ProbeType, nil)

	prBytes, _ := pr.MarshalBinary()

	s, mock := newRosterMock()
	mock.ExpectQuery(`SELECT contact, jid, presence FROM roster_notifications WHERE contact = \$1`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow(
				"ortuman",
				"noelia@jackal.im",
				prBytes,
			),
		)

	// when
	rns, err := s.FetchRosterNotifications(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Len(t, rns, 1)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLRoster_FetchRosterGroups(t *testing.T) {
	// given
	cols := []string{`DISTINCT UNNEST(groups)`}

	s, mock := newRosterMock()
	mock.ExpectQuery(`SELECT DISTINCT UNNEST\(groups\) FROM roster_items WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows(cols).
				AddRow("VIP").
				AddRow("Buddies"),
		)

	// when
	groups, err := s.FetchRosterGroups(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Len(t, groups, 2)

	require.Nil(t, mock.ExpectationsWereMet())
}

func newRosterMock() (*pgSQLRosterRep, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLRosterRep{conn: s}, sqlMock
}
