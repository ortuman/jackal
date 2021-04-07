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
	blocklistmodel "github.com/ortuman/jackal/model/blocklist"
	"github.com/stretchr/testify/require"
)

func TestPgSQLBlockList_Upsert(t *testing.T) {
	s, mock := newBlockListMock()
	mock.ExpectExec(`INSERT INTO blocklist_items \(username,jid\) VALUES \(\$1,\$2\) ON CONFLICT \(username, jid\) DO NOTHING`).
		WithArgs("ortuman", "noelia@jackal.im").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.UpsertBlockListItem(context.Background(), &blocklistmodel.Item{
		Username: "ortuman",
		JID:      "noelia@jackal.im",
	})
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLBlockList_Fetch(t *testing.T) {
	var blockListColumns = []string{"username", "jid"}
	s, mock := newBlockListMock()
	mock.ExpectQuery(`SELECT username, jid FROM blocklist_items WHERE username = \$1 ORDER BY created_at`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows(blockListColumns).AddRow("ortuman", "noelia@jackal.im"),
		)

	_, err := s.FetchBlockListItems(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLBlockList_DeleteItem(t *testing.T) {
	s, mock := newBlockListMock()
	mock.ExpectExec(`DELETE FROM blocklist_items WHERE \(username = \$1 AND jid = \$2\)`).
		WithArgs("ortuman", "noelia@jackal.im").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeleteBlockListItem(context.Background(), &blocklistmodel.Item{Username: "ortuman", JID: "noelia@jackal.im"})
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestPgSQLBlockList_DeleteItems(t *testing.T) {
	s, mock := newBlockListMock()
	mock.ExpectExec(`DELETE FROM blocklist_items WHERE username = \$1`).
		WithArgs("ortuman").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeleteBlockListItems(context.Background(), "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func newBlockListMock() (*pgSQLBlockListRep, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLBlockListRep{conn: s}, sqlMock
}
