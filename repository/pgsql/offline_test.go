// Copyright 2021 The jackal Authors
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
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/stretchr/testify/require"
)

func TestPgSQLOffline_InsertOfflineMessage(t *testing.T) {
	// given
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage(true)

	msgBytes, _ := msg.MarshalBinary()

	s, mock := newOfflineMock()
	mock.ExpectExec(`INSERT INTO offline_messages \(username,message\) VALUES \(\$1,\$2\)`).
		WithArgs("ortuman", msgBytes).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.InsertOfflineMessage(context.Background(), msg, "ortuman")

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLOffline_CountOfflineMessage(t *testing.T) {
	// given
	s, mock := newOfflineMock()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM offline_messages WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(10),
		)

	// when
	c, err := s.CountOfflineMessages(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Equal(t, 10, c)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLOffline_FetchOfflineMessage(t *testing.T) {
	// given
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage(true)

	msgBytes, _ := msg.MarshalBinary()

	s, mock := newOfflineMock()
	mock.ExpectQuery(`SELECT message FROM offline_messages WHERE username = \$1 ORDER BY id`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows([]string{"message"}).AddRow(msgBytes),
		)

	// when
	ms, err := s.FetchOfflineMessages(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Len(t, ms, 1)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLOffline_DeleteOfflineMessage(t *testing.T) {
	// given
	s, mock := newOfflineMock()
	mock.ExpectExec(`DELETE FROM offline_messages WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.DeleteOfflineMessages(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func newOfflineMock() (*pgSQLOfflineRep, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLOfflineRep{conn: s}, sqlMock
}
