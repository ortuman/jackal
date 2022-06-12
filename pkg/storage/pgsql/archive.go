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
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	kitlog "github.com/go-kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/jackal-xmpp/stravaganza"
	"github.com/jackal-xmpp/stravaganza/jid"
	archivemodel "github.com/ortuman/jackal/pkg/model/archive"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	archiveTableName = "archives"

	archiveStampFormat = "2006-01-02T15:04:05Z"
)

type pgSQLArchiveRep struct {
	conn   conn
	logger kitlog.Logger
}

func (r *pgSQLArchiveRep) InsertArchiveMessage(ctx context.Context, message *archivemodel.Message) error {
	b, err := proto.Marshal(message.Message)
	if err != nil {
		return err
	}
	fromJID, _ := jid.NewWithString(message.FromJid, true)
	toJID, _ := jid.NewWithString(message.ToJid, true)

	q := sq.Insert(archiveTableName).
		Prefix(noLoadBalancePrefix).
		Columns("archive_id", "id", `"from"`, "from_bare", `"to"`, "to_bare", "message").
		Values(
			message.ArchiveId,
			message.Id,
			fromJID.String(),
			fromJID.ToBareJID().String(),
			toJID.String(),
			toJID.ToBareJID().String(),
			b,
		)

	_, err = q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLArchiveRep) FetchArchiveMetadata(ctx context.Context, archiveID string) (*archivemodel.Metadata, error) {
	fromExpr := `FROM `
	fromExpr += `(SELECT "id", created_at FROM archives WHERE serial = (SELECT MIN(serial) FROM archives WHERE archive_id = $1)) AS min,`
	fromExpr += `(SELECT "id", created_at FROM archives WHERE serial = (SELECT MAX(serial) FROM archives WHERE archive_id = $1)) AS max`

	q := sq.Select("min.id, min.created_at, max.id, max.created_at").Suffix(fromExpr, archiveID)

	var start, end time.Time
	var metadata archivemodel.Metadata

	err := q.RunWith(r.conn).
		QueryRowContext(ctx).
		Scan(
			&metadata.StartId,
			&start,
			&metadata.EndId,
			&end,
		)

	switch err {
	case nil:
		metadata.StartTimestamp = start.UTC().Format(archiveStampFormat)
		metadata.EndTimestamp = end.UTC().Format(archiveStampFormat)
		return &metadata, nil

	case sql.ErrNoRows:
		return nil, nil

	default:
		return nil, err
	}
}

func (r *pgSQLArchiveRep) FetchArchiveMessages(ctx context.Context, f *archivemodel.Filters, archiveID string) ([]*archivemodel.Message, error) {
	q := sq.Select("id", `"from"`, `"to"`, "message", "created_at").
		From(archiveTableName).
		Where(filtersToPred(f, archiveID)).
		OrderBy("created_at").
		PlaceholderFormat(sq.Dollar)

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, r.logger)

	retVal, err := scanArchiveMessages(rows, archiveID)
	if err != nil {
		return nil, err
	}
	return retVal, err
}

func (r *pgSQLArchiveRep) DeleteArchiveOldestMessages(ctx context.Context, archiveID string, maxElements int) error {
	q := sq.Delete(archiveTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.And{
			sq.Eq{"archive_id": archiveID},
			sq.Expr(`"id" NOT IN (SELECT "id" FROM archives WHERE archive_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET 0)`, archiveID, maxElements),
		})
	_, err := q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLArchiveRep) DeleteArchive(ctx context.Context, archiveID string) error {
	q := sq.Delete(archiveTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.Eq{"archive_id": archiveID})
	_, err := q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func filtersToPred(f *archivemodel.Filters, archiveID string) (interface{}, error) {
	pred := sq.And{
		sq.Eq{"archive_id": archiveID},
	}
	// filtering by JID
	if len(f.With) > 0 {
		jd, err := jid.NewWithString(f.With, false)
		if err != nil {
			return nil, err
		}
		switch {
		case jd.IsFull():
			pred = append(pred, sq.Expr(`("to" = ? OR "from" = ?)`, jd.String(), jd.String()))

		default:
			pred = append(pred, sq.Expr(`(to_bare = ? OR from_bare = ?)`, jd.String(), jd.String()))
		}
	}

	// filtering by id
	if len(f.Ids) > 0 {
		pred = append(pred, sq.Eq{"id": f.Ids})
	} else {
		if len(f.BeforeId) > 0 {
			pred = append(pred, sq.Expr(`(serial < (SELECT serial FROM archives WHERE "id" = ? AND archive_id = ?))`, f.BeforeId, archiveID))
		}
		if len(f.AfterId) > 0 {
			pred = append(pred, sq.Expr(`(serial > (SELECT serial FROM archives WHERE "id" = ? AND archive_id = ?))`, f.AfterId, archiveID))
		}
	}

	// filtering by timestamp
	if f.Start != nil {
		pred = append(pred, sq.Expr("EXTRACT(epoch FROM created_at) > ?", f.Start.GetSeconds()))
	}
	if f.End != nil {
		pred = append(pred, sq.Expr("EXTRACT(epoch FROM created_at) < ?", f.End.GetSeconds()))
	}
	return pred, nil
}

func scanArchiveMessages(scanner rowsScanner, archiveID string) ([]*archivemodel.Message, error) {
	var ret []*archivemodel.Message
	for scanner.Next() {
		msg, err := scanArchiveMessage(scanner, archiveID)
		if err != nil {
			return nil, err
		}
		ret = append(ret, msg)
	}
	return ret, nil
}

func scanArchiveMessage(scanner rowsScanner, archiveID string) (*archivemodel.Message, error) {
	var ret archivemodel.Message

	var b []byte
	var tm time.Time

	if err := scanner.Scan(&ret.Id, &ret.FromJid, &ret.ToJid, &b, &tm); err != nil {
		return nil, err
	}
	sb, err := stravaganza.NewBuilderFromBinary(b)
	if err != nil {
		return nil, err
	}
	msg, err := sb.BuildMessage()
	if err != nil {
		return nil, err
	}
	ret.ArchiveId = archiveID
	ret.Message = msg.Proto()
	ret.Stamp = timestamppb.New(tm)

	return &ret, nil
}
