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
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackal-xmpp/stravaganza/v2"
)

const (
	vCardsTableName = "vcards"
)

type pgSQLVCardRep struct {
	conn conn
}

func (r *pgSQLVCardRep) UpsertVCard(ctx context.Context, vCard stravaganza.Element, username string) error {
	b, err := vCard.MarshalBinary()
	if err != nil {
		return err
	}
	q := sq.Insert(vCardsTableName).
		Columns("username", "vcard").
		Values(username, b).
		Suffix("ON CONFLICT (username) DO UPDATE SET vcard = $2")

	_, err = q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLVCardRep) FetchVCard(ctx context.Context, username string) (stravaganza.Element, error) {
	q := sq.Select("vcard").
		From(vCardsTableName).
		Where(sq.Eq{"username": username})

	var vCardB []byte
	err := q.RunWith(r.conn).
		QueryRowContext(ctx).
		Scan(&vCardB)
	switch err {
	case nil:
		b, err := stravaganza.NewBuilderFromBinary(vCardB)
		if err != nil {
			return nil, err
		}
		return b.Build(), nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (r *pgSQLVCardRep) DeleteVCard(ctx context.Context, username string) error {
	_, err := sq.Delete(vCardsTableName).
		Where(sq.Eq{"username": username}).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}
