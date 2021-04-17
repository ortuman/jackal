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

func TestPgSQLPrivate_FetchPrivate(t *testing.T) {
	// given
	prv := testPrivate()
	b, _ := prv.MarshalBinary()

	s, mock := newPrivateMock()
	mock.ExpectQuery(`SELECT data FROM private_storage WHERE \(namespace = \$1 AND username = \$2\)`).
		WithArgs("exodus:prefs", "ortuman").
		WillReturnRows(
			sqlmock.NewRows([]string{"data"}).AddRow(b),
		)

	// when
	prv, err := s.FetchPrivate(context.Background(), "exodus:prefs", "ortuman")

	// then
	require.Nil(t, err)
	require.NotNil(t, prv)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLPrivate_UpsertPrivate(t *testing.T) {
	// given
	prv := testPrivate()
	b, _ := prv.MarshalBinary()

	s, mock := newPrivateMock()
	mock.ExpectExec(`INSERT INTO private_storage \(username,namespace,data\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \(username, namespace\) DO UPDATE SET data = \$3`).
		WithArgs("ortuman", "exodus:prefs", b).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.UpsertPrivate(context.Background(), prv, "exodus:prefs", "ortuman")

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLPrivate_DeletePrivates(t *testing.T) {
	s, mock := newPrivateMock()

	mock.ExpectExec(`DELETE FROM private_storage WHERE username = \$1`).
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeletePrivates(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func newPrivateMock() (*pgSQLPrivateRep, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLPrivateRep{conn: s}, sqlMock
}

func testPrivate() stravaganza.Element {
	return stravaganza.NewBuilder("exodus").
		WithAttribute(stravaganza.Namespace, "exodus:prefs").
		WithChild(
			stravaganza.NewBuilder("defaultnick").WithText("Hamlet").Build(),
		).
		Build()
}
