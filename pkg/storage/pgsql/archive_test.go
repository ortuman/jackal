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
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/protobuf/proto"
	"github.com/jackal-xmpp/stravaganza"
	archivemodel "github.com/ortuman/jackal/pkg/model/archive"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestPgSQLArchive_InsertArchiveMessage(t *testing.T) {
	// given
	b := stravaganza.NewMessageBuilder()
	b.WithAttribute("from", "noelia@jackal.im/yard")
	b.WithAttribute("to", "ortuman@jackal.im/balcony")
	b.WithChild(
		stravaganza.NewBuilder("body").
			WithText("I'll give thee a wind.").
			Build(),
	)
	msg, _ := b.BuildMessage()

	aMsg := &archivemodel.Message{
		ArchiveId: "ortuman",
		Id:        "id1234",
		FromJid:   "ortuman@jackal.im/local",
		ToJid:     "ortuman@jabber.org/remote",
		Message:   msg.Proto(),
	}
	msgBytes, _ := proto.Marshal(aMsg.Message)

	s, mock := newArchiveMock()
	mock.ExpectExec(`INSERT INTO archives \(archive_id,id,"from",from_bare,"to",to_bare,message\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6,\$7\)`).
		WithArgs("ortuman", "id1234", "ortuman@jackal.im/local", "ortuman@jackal.im", "ortuman@jabber.org/remote", "ortuman@jabber.org", msgBytes).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.InsertArchiveMessage(context.Background(), aMsg)

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLArchive_FetchArchiveMetadata(t *testing.T) {
	minT := time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)
	maxT := time.Date(2022, 12, 12, 00, 00, 00, 00, time.UTC)

	// given
	s, mock := newArchiveMock()
	mock.ExpectQuery(`SELECT min.id, min.created_at, max.id, max.created_at FROM \(SELECT "id", created_at FROM archives WHERE serial = \(SELECT MIN\(serial\) FROM archives WHERE archive_id = \$1\)\) AS min,\(SELECT "id", created_at FROM archives WHERE serial = \(SELECT MAX\(serial\) FROM archives WHERE archive_id = \$1\)\) AS max`).
		WithArgs("ortuman").
		WillReturnRows(
			sqlmock.NewRows([]string{"min.id", "min.created_at", "max.id", "max.created_at"}).AddRow("YWxwaGEg", minT, "b21lZ2Eg", maxT),
		)

	// when
	metadata, err := s.FetchArchiveMetadata(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.NotNil(t, metadata)

	require.Equal(t, "YWxwaGEg", metadata.StartId)
	require.Equal(t, "2022-01-01T00:00:00Z", metadata.StartTimestamp)
	require.Equal(t, "b21lZ2Eg", metadata.EndId)
	require.Equal(t, "2022-12-12T00:00:00Z", metadata.EndTimestamp)

	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLArchive_FetchArchiveMessages(t *testing.T) {
	starTm := time.Date(2022, time.July, 6, 14, 7, 43, 167051000, time.UTC)
	endTm := time.Date(2023, time.July, 7, 15, 7, 43, 167051000, time.UTC)

	tcs := map[string]struct {
		filters     *archivemodel.Filters
		withArgs    []driver.Value
		expectQuery string
	}{
		"by bare jid": {
			filters:     &archivemodel.Filters{With: "noelia@jackal.im"},
			withArgs:    []driver.Value{"ortuman", "noelia@jackal.im", "noelia@jackal.im"},
			expectQuery: `SELECT id, "from", "to", message, created_at FROM archives WHERE \(archive_id = \$1 AND \(to_bare = \$2 OR from_bare = \$3\)\) ORDER BY created_at`,
		},
		"by full jid": {
			filters:     &archivemodel.Filters{With: "noelia@jackal.im/yard"},
			withArgs:    []driver.Value{"ortuman", "noelia@jackal.im/yard", "noelia@jackal.im/yard"},
			expectQuery: `SELECT id, "from", "to", message, created_at FROM archives WHERE \(archive_id = \$1 AND \("to" = \$2 OR "from" = \$3\)\) ORDER BY created_at`,
		},
		"by ids": {
			filters:     &archivemodel.Filters{Ids: []string{"id1234", "id5678"}},
			withArgs:    []driver.Value{"ortuman", "id1234", "id5678"},
			expectQuery: `SELECT id, "from", "to", message, created_at FROM archives WHERE \(archive_id = \$1 AND id IN \(\$2,\$3\)\) ORDER BY created_at`,
		},
		"by before id": {
			filters:     &archivemodel.Filters{BeforeId: "id1234"},
			withArgs:    []driver.Value{"ortuman", "id1234", "ortuman"},
			expectQuery: `SELECT id, "from", "to", message, created_at FROM archives WHERE \(archive_id = \$1 AND \(serial < \(SELECT serial FROM archives WHERE "id" = \$2 AND archive_id = \$3\)\)\) ORDER BY created_at`,
		},
		"by after id": {
			filters:     &archivemodel.Filters{AfterId: "id1234"},
			withArgs:    []driver.Value{"ortuman", "id1234", "ortuman"},
			expectQuery: `SELECT id, "from", "to", message, created_at FROM archives WHERE \(archive_id = \$1 AND \(serial > \(SELECT serial FROM archives WHERE "id" = \$2 AND archive_id = \$3\)\)\) ORDER BY created_at`,
		},
		"by before and after id": {
			filters:     &archivemodel.Filters{BeforeId: "id1234", AfterId: "id5678"},
			withArgs:    []driver.Value{"ortuman", "id1234", "ortuman", "id5678", "ortuman"},
			expectQuery: `SELECT id, "from", "to", message, created_at FROM archives WHERE \(archive_id = \$1 AND \(serial < \(SELECT serial FROM archives WHERE "id" = \$2 AND archive_id = \$3\)\) AND \(serial > \(SELECT serial FROM archives WHERE "id" = \$4 AND archive_id = \$5\)\)\) ORDER BY created_at`,
		},
		"by start timestamp": {
			filters:     &archivemodel.Filters{Start: timestamppb.New(starTm)},
			withArgs:    []driver.Value{"ortuman", toEpoch(timestamppb.New(starTm))},
			expectQuery: `SELECT id, "from", "to", message, created_at FROM archives WHERE \(archive_id = \$1 AND EXTRACT\(epoch FROM created_at\) > \$2\) ORDER BY created_at`,
		},
		"by end timestamp": {
			filters:     &archivemodel.Filters{End: timestamppb.New(endTm)},
			withArgs:    []driver.Value{"ortuman", toEpoch(timestamppb.New(endTm))},
			expectQuery: `SELECT id, "from", "to", message, created_at FROM archives WHERE \(archive_id = \$1 AND EXTRACT\(epoch FROM created_at\) < \$2\) ORDER BY created_at`,
		},
		"by start and end timestamp": {
			filters:     &archivemodel.Filters{Start: timestamppb.New(starTm), End: timestamppb.New(endTm)},
			withArgs:    []driver.Value{"ortuman", toEpoch(timestamppb.New(starTm)), toEpoch(timestamppb.New(endTm))},
			expectQuery: `SELECT id, "from", "to", message, created_at FROM archives WHERE \(archive_id = \$1 AND EXTRACT\(epoch FROM created_at\) > \$2 AND EXTRACT\(epoch FROM created_at\) < \$3\) ORDER BY created_at`,
		},
	}
	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			b := stravaganza.NewMessageBuilder()
			b.WithAttribute("from", "noelia@jackal.im/yard")
			b.WithAttribute("to", "ortuman@jackal.im/balcony")
			b.WithChild(
				stravaganza.NewBuilder("body").
					WithText("I'll give thee a wind.").
					Build(),
			)
			msg, _ := b.BuildMessage()

			msgBytes, _ := msg.MarshalBinary()
			tmNow := time.Date(2022, time.July, 6, 14, 7, 43, 167051000, time.UTC)

			rows := sqlmock.NewRows([]string{"id", "from", "to", "message", "created_at"}).
				AddRow("id1234", "ortuman@jackal.im", "noelia@jackal.im", msgBytes, tmNow)

			s, mock := newArchiveMock()
			mock.ExpectQuery(tc.expectQuery).
				WithArgs(tc.withArgs...).
				WillReturnRows(rows)

			// when
			messages, err := s.FetchArchiveMessages(context.Background(), tc.filters, "ortuman")

			require.NoError(t, err)
			require.Nil(t, mock.ExpectationsWereMet())

			// then
			require.Len(t, messages, 1)
			require.Equal(t, "id1234", messages[0].Id)
			require.Equal(t, tmNow, messages[0].Stamp.AsTime())
		})
	}
}

func TestPgSQLArchive_DeleteArchiveOldestMessages(t *testing.T) {
	// given
	s, mock := newArchiveMock()
	mock.ExpectExec(`DELETE FROM archives WHERE \(archive_id = \$1 AND "id" NOT IN \(SELECT "id" FROM archives WHERE archive_id = \$2 ORDER BY created_at DESC LIMIT \$3 OFFSET 0\)\)`).
		WithArgs("ortuman", "ortuman", 1234).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.DeleteArchiveOldestMessages(context.Background(), "ortuman", 1234)

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func TestPgSQLArchive_DeleteArchive(t *testing.T) {
	// given
	s, mock := newArchiveMock()
	mock.ExpectExec(`DELETE FROM archives WHERE archive_id = \$1`).
		WithArgs("ortuman").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// when
	err := s.DeleteArchive(context.Background(), "ortuman")

	// then
	require.Nil(t, err)
	require.Nil(t, mock.ExpectationsWereMet())
}

func newArchiveMock() (*pgSQLArchiveRep, sqlmock.Sqlmock) {
	s, sqlMock := newPgSQLMock()
	return &pgSQLArchiveRep{conn: s}, sqlMock
}
