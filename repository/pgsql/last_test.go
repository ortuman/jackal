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
	lastmodel "github.com/ortuman/jackal/model/last"
	"github.com/stretchr/testify/require"
)

func TestPgSQLLast_Upsert(t *testing.T) {
	// given
	s, mock := newLastMock()
	mock.ExpectExec(`INSERT INTO last \(username,seconds,status\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \(username\) DO UPDATE SET seconds = \$2, status = \$3`).
		WithArgs("ortuman", 1234, "Heading home").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.UpsertLast(context.Background(), &lastmodel.Last{
		Username: "ortuman",
		Seconds:  1234,
		Status:   "Heading home",
	})

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLLast_Fetch(t *testing.T) {
	// given
	var lastColumns = []string{"username", "seconds", "status"}
	s, mock := newLastMock()
	mock.ExpectQuery(`SELECT username, seconds, status FROM last WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows(lastColumns).AddRow("ortuman", 1234, "Heading home"),
		)

	// when
	last, err := s.FetchLast(context.Background(), "ortuman")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, last.Username, "ortuman")
	require.Equal(t, last.Seconds, int64(1234))
	require.Equal(t, last.Status, "Heading home")
}

func TestPgSQLLast_Delete(t *testing.T) {
	// given
	s, mock := newLastMock()
	mock.ExpectExec(`DELETE FROM last WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.DeleteLast(context.Background(), "ortuman")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func newLastMock() (*pgSQLLastRep, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLLastRep{conn: s}, sqlMock
}
