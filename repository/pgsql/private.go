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
	"github.com/jackal-xmpp/stravaganza"
)

const privateStorageTableName = "private_storage"

type pgSQLPrivateRep struct {
	conn conn
}

func (r *pgSQLPrivateRep) FetchPrivate(ctx context.Context, namespace, username string) (stravaganza.Element, error) {
	q := sq.Select("data").
		From(privateStorageTableName).
		Where(sq.And{sq.Eq{"namespace": namespace}, sq.Eq{"username": username}})

	var b []byte
	err := q.RunWith(r.conn).QueryRowContext(ctx).Scan(&b)
	switch err {
	case nil:
		pb, err := stravaganza.NewBuilderFromBinary(b)
		if err != nil {
			return nil, err
		}
		return pb.Build(), nil

	case sql.ErrNoRows:
		return nil, nil

	default:
		return nil, err
	}
}

func (r *pgSQLPrivateRep) UpsertPrivate(ctx context.Context, private stravaganza.Element, namespace, username string) error {
	b, err := private.MarshalBinary()
	if err != nil {
		return err
	}
	q := sq.Insert(privateStorageTableName).
		Columns("username", "namespace", "data").
		Values(username, namespace, b).
		Suffix("ON CONFLICT (username, namespace) DO UPDATE SET data = $3")

	_, err = q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLPrivateRep) DeletePrivates(ctx context.Context, username string) error {
	_, err := sq.Delete(privateStorageTableName).
		Where(sq.Eq{"username": username}).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}
