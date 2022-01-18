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

	kitlog "github.com/go-kit/log"

	usermodel "github.com/ortuman/jackal/pkg/model/user"

	sq "github.com/Masterminds/squirrel"
)

const (
	usersTableName = "users"
)

type pgSQLUserRep struct {
	conn   conn
	logger kitlog.Logger
}

func (r *pgSQLUserRep) UpsertUser(ctx context.Context, user *usermodel.User) error {
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
		user.Scram.Sha1,
		user.Scram.Sha256,
		user.Scram.Sha512,
		user.Scram.Sha3512,
		user.Scram.Salt,
		user.Scram.IterationCount,
		user.Scram.PepperId,
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

func (r *pgSQLUserRep) FetchUser(ctx context.Context, username string) (*usermodel.User, error) {
	var usr usermodel.User
	usr.Scram = &usermodel.Scram{}

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
			&usr.Scram.Sha1,
			&usr.Scram.Sha256,
			&usr.Scram.Sha512,
			&usr.Scram.Sha3512,
			&usr.Scram.Salt,
			&usr.Scram.IterationCount,
			&usr.Scram.PepperId,
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
