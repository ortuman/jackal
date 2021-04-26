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
	coremodel "github.com/ortuman/jackal/pkg/model/core"
)

const (
	usersTableName = "users"
)

type pgSQLUserRep struct {
	conn conn
}

func (r *pgSQLUserRep) UpsertUser(ctx context.Context, user *coremodel.User) error {
	cols := []string{
		"username",
		"h_sha_1",
		"h_sha_256",
		"h_sha_512",
		"h_sha3_512",
		"salt",
		"iteration_count",
		"pepper_id",
	}
	vals := []interface{}{
		user.Username,
		user.Scram.SHA1,
		user.Scram.SHA256,
		user.Scram.SHA512,
		user.Scram.SHA3512,
		user.Scram.Salt,
		user.Scram.IterationCount,
		user.Scram.PepperID,
	}
	q := sq.Insert(usersTableName).
		Columns(cols...).
		Values(vals...).
		Suffix("ON CONFLICT (username) DO UPDATE SET h_sha_1 = $2, h_sha_256 = $3, h_sha_512 = $4, h_sha3_512 = $5, salt = $6, iteration_count = $7, pepper_id = $8")

	_, err := q.RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLUserRep) DeleteUser(ctx context.Context, username string) error {
	_, err := sq.Delete(usersTableName).
		Where(sq.Eq{"username": username}).
		RunWith(r.conn).
		ExecContext(ctx)
	return err
}

func (r *pgSQLUserRep) FetchUser(ctx context.Context, username string) (*coremodel.User, error) {
	var usr coremodel.User

	cols := []string{
		"username",
		"h_sha_1",
		"h_sha_256",
		"h_sha_512",
		"h_sha3_512",
		"salt",
		"iteration_count",
		"pepper_id",
	}
	q := sq.Select(cols...).
		From(usersTableName).
		Where(sq.Eq{"username": username})

	err := q.RunWith(r.conn).
		QueryRowContext(ctx).
		Scan(
			&usr.Username,
			&usr.Scram.SHA1,
			&usr.Scram.SHA256,
			&usr.Scram.SHA512,
			&usr.Scram.SHA3512,
			&usr.Scram.Salt,
			&usr.Scram.IterationCount,
			&usr.Scram.PepperID,
		)
	switch err {
	case nil:
		return &usr, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (r *pgSQLUserRep) UserExists(ctx context.Context, username string) (bool, error) {
	q := sq.Select("COUNT(*)").
		From(usersTableName).
		Where(sq.Eq{"username": username})

	var count int
	err := q.RunWith(r.conn).QueryRowContext(ctx).Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}
