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
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	lastmodel "github.com/ortuman/jackal/model/last"
)

const (
	lastTableName = "last"
)

type pgSQLLastRep struct {
	conn conn
}

func (r *pgSQLLastRep) UpsertLast(ctx context.Context, last *lastmodel.Last) error {
	_, err := sq.Insert(lastTableName).
		Columns("username", "seconds", "status").
		Values(last.Username, last.Seconds, last.Status).
		Suffix("ON CONFLICT (username) DO UPDATE SET seconds = $2, status = $3").
		RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLLastRep) FetchLast(ctx context.Context, username string) (*lastmodel.Last, error) {
	q := sq.Select("username", "seconds", "status").
		From(lastTableName).
		Where(sq.Eq{"username": username})

	var last lastmodel.Last
	err := q.RunWith(r.conn).
		QueryRowContext(ctx).
		Scan(&last.Username, &last.Seconds, &last.Status)
	switch err {
	case nil:
		return &last, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (r *pgSQLLastRep) DeleteLast(ctx context.Context, username string) error {
	_, err := sq.Delete(lastTableName).
		Where(sq.Eq{"username": username}).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}
