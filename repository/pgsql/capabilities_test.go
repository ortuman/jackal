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
	"github.com/lib/pq"
	capsmodel "github.com/ortuman/jackal/model/caps"
	"github.com/stretchr/testify/require"
)

func TestPgSQLCapabilitiesRep_UpsertCapabilities(t *testing.T) {
	// given
	cp := &capsmodel.Capabilities{
		Node:     "n0",
		Ver:      "v0",
		Features: []string{"f100"},
	}
	s, mock := newCapabilitiesMock()
	mock.ExpectExec(`INSERT INTO capabilities \(node,ver,features\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \(node, ver\) DO UPDATE SET features = \$3`).
		WithArgs(cp.Node, cp.Ver, pq.Array(cp.Features)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.UpsertCapabilities(context.Background(), cp)

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLCapabilitiesRep_FetchCapabilities(t *testing.T) {
	// given
	s, mock := newCapabilitiesMock()
	mock.ExpectQuery(`SELECT node, ver, features FROM capabilities WHERE \(node = \$1 AND ver = \$2\)`).
		WithArgs("n0", "v0").
		WillReturnRows(sqlmock.NewRows([]string{"node", "ver", "features"}).
			AddRow("n0", "v0", pq.Array([]string{"f100"})),
		)

	// when
	caps, err := s.FetchCapabilities(context.Background(), "n0", "v0")

	// then
	require.Nil(t, err)
	require.Len(t, caps.Features, 1)

	require.Nil(t, mock.ExpectationsWereMet())
}

func newCapabilitiesMock() (*pgSQLCapabilitiesRep, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLCapabilitiesRep{conn: s}, sqlMock
}
