// Copyright 2022 The jackal Authors
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
	"github.com/stretchr/testify/require"
)

func TestPgSQLLocker_Lock(t *testing.T) {
	// given
	s, mock := newLockerMock()

	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(hashtext\(\$1\)\)`).
		WithArgs("l1").
		WillReturnRows(
			sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true),
		)

	// when
	err := s.Lock(context.Background(), "l1")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLLocker_Unlock(t *testing.T) {
	// given
	s, mock := newLockerMock()

	mock.ExpectExec(`SELECT pg_advisory_unlock\(hashtext\(\$1\)\)`).
		WithArgs("l1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.Unlock(context.Background(), "l1")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func newLockerMock() (*pgSQLLocker, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLLocker{conn: s}, sqlMock
}
