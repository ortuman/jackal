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
	"github.com/jackal-xmpp/stravaganza/v2"
	"github.com/stretchr/testify/require"
)

func TestPgSQLVCard_Upsert(t *testing.T) {
	// given
	vcEl := stravaganza.NewBuilder("vcard").Build()
	b, _ := vcEl.MarshalBinary()

	s, mock := newVCardMock()
	mock.ExpectExec(`INSERT INTO vcards \(username,vcard\) VALUES \(\$1,\$2\) ON CONFLICT \(username\) DO UPDATE SET vcard = \$2`).
		WithArgs("ortuman", b).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.UpsertVCard(context.Background(), vcEl, "ortuman")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLVCard_Fetch(t *testing.T) {
	// given
	vcEl := stravaganza.NewBuilder("vcard").Build()
	b, _ := vcEl.MarshalBinary()

	var lastColumns = []string{"vcard"}
	s, mock := newVCardMock()
	mock.ExpectQuery(`SELECT vcard FROM vcards WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows(lastColumns).AddRow(b),
		)

	// when
	vc, err := s.FetchVCard(context.Background(), "ortuman")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.NotNil(t, vc)
	require.Equal(t, vc.String(), vcEl.String())
}

func TestPgSQLVCard_Delete(t *testing.T) {
	// given
	s, mock := newVCardMock()
	mock.ExpectExec(`DELETE FROM vcards WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// when
	err := s.DeleteVCard(context.Background(), "ortuman")

	// then
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func newVCardMock() (*pgSQLVCardRep, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLVCardRep{conn: s}, sqlMock
}
