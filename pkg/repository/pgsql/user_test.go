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
	coremodel "github.com/ortuman/jackal/pkg/model/core"
	"github.com/stretchr/testify/require"
)

func TestPgSQLUser_Upsert(t *testing.T) {
	s, mock := newUserMock()
	mock.ExpectExec(`INSERT INTO users \(username,h_sha_1,h_sha_256,h_sha_512,h_sha3_512,salt,iteration_count,pepper_id\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6,\$7,\$8\) ON CONFLICT \(username\) DO UPDATE SET h_sha_1 = \$2, h_sha_256 = \$3, h_sha_512 = \$4, h_sha3_512 = \$5, salt = \$6, iteration_count = \$7, pepper_id = \$8`).
		WithArgs("ortuman", "v_sha_1", "v_sha_256", "v_sha_512", "v_sha3_512", "salt", 1024, "v1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	usr := coremodel.User{Username: "ortuman"}
	usr.Scram.SHA1 = "v_sha_1"
	usr.Scram.SHA256 = "v_sha_256"
	usr.Scram.SHA512 = "v_sha_512"
	usr.Scram.SHA3512 = "v_sha3_512"
	usr.Scram.Salt = "salt"
	usr.Scram.IterationCount = 1024
	usr.Scram.PepperID = "v1"

	err := s.UpsertUser(context.Background(), &usr)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLUser_Fetch(t *testing.T) {
	cols := []string{
		"username",
		"h_sha_1",
		"h_sha_256",
		"h_sha_512",
		"h_sha3_512",
		"salt",
		"iteration_count",
		"pepper_id",
	}

	s, mock := newUserMock()
	mock.ExpectQuery(`SELECT username, h_sha_1, h_sha_256, h_sha_512, h_sha3_512, salt, iteration_count, pepper_id FROM users WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows(cols).AddRow("ortuman", "v_sha_1", "v_sha_256", "v_sha_512", "v_sha3_512", "salt", 1024, "v1"),
		)

	usr, err := s.FetchUser(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.NotNil(t, usr)

	require.Equal(t, "ortuman", usr.Username)
	require.Equal(t, "v_sha_1", usr.Scram.SHA1)
	require.Equal(t, "v_sha_256", usr.Scram.SHA256)
	require.Equal(t, "v_sha_512", usr.Scram.SHA512)
	require.Equal(t, "v_sha3_512", usr.Scram.SHA3512)
	require.Equal(t, "salt", usr.Scram.Salt)
	require.Equal(t, 1024, usr.Scram.IterationCount)
	require.Equal(t, "v1", usr.Scram.PepperID)
}

func TestPgSQLUser_Delete(t *testing.T) {
	s, mock := newUserMock()

	mock.ExpectExec(`DELETE FROM users WHERE username = \$1`).
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeleteUser(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLUser_Exists(t *testing.T) {
	countCols := []string{"count"}

	s, mock := newUserMock()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows(countCols).AddRow(1),
		)

	ok, err := s.UserExists(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.True(t, ok)
}

func newUserMock() (*pgSQLUserRep, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLUserRep{conn: s}, sqlMock
}
