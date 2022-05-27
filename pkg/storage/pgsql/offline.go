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

	sq "github.com/Masterminds/squirrel"
	kitlog "github.com/go-kit/log"
	"github.com/jackal-xmpp/stravaganza"
)

const offlineMessagesTableName = "offline_messages"

type pgSQLOfflineRep struct {
	conn   conn
	logger kitlog.Logger
}

func (r *pgSQLOfflineRep) InsertOfflineMessage(ctx context.Context, message *stravaganza.Message, username string) error {
	b, err := message.MarshalBinary()
	if err != nil {
		return err
	}
	q := sq.Insert(offlineMessagesTableName).
		Prefix(noLoadBalancePrefix).
		Columns("username", "message").
		Values(username, b)

	_, err = q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLOfflineRep) CountOfflineMessages(ctx context.Context, username string) (int, error) {
	var count int

	q := sq.Select("COUNT(*)").
		From(offlineMessagesTableName).
		Where(sq.Eq{"username": username})

	if err := q.RunWith(r.conn).QueryRowContext(ctx).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *pgSQLOfflineRep) FetchOfflineMessages(ctx context.Context, username string) ([]*stravaganza.Message, error) {
	q := sq.Select("message").
		From(offlineMessagesTableName).
		Where(sq.Eq{"username": username}).
		OrderBy("id")

	rows, err := q.RunWith(r.conn).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, r.logger)

	var ms []*stravaganza.Message
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
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
		ms = append(ms, msg)
	}
	return ms, nil
}

func (r *pgSQLOfflineRep) DeleteOfflineMessages(ctx context.Context, username string) error {
	q := sq.Delete(offlineMessagesTableName).
		Prefix(noLoadBalancePrefix).
		Where(sq.Eq{"username": username})
	_, err := q.RunWith(r.conn).ExecContext(ctx)
	return err
}
