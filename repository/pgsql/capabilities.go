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
	"github.com/lib/pq"
	capsmodel "github.com/ortuman/jackal/model/caps"
)

const (
	capsTableName = "capabilities"
)

type pgSQLCapabilitiesRep struct {
	conn conn
}

func (r *pgSQLCapabilitiesRep) UpsertCapabilities(ctx context.Context, caps *capsmodel.Capabilities) error {
	var b []byte
	var err error

	if caps.Form != nil {
		b, err = caps.Form.MarshalBinary()
		if err != nil {
			return err
		}
	}
	_, err = sq.Insert(capsTableName).
		Columns("node", "ver", "features", "form").
		Values(caps.Node, caps.Ver, pq.Array(caps.Features), b).
		Suffix("ON CONFLICT (node, ver) DO UPDATE SET features = $3, form = $4").
		RunWith(r.conn).ExecContext(ctx)
	return err
}

func (r *pgSQLCapabilitiesRep) CapabilitiesExist(ctx context.Context, node, ver string) (bool, error) {
	var count int
	row := sq.Select("COUNT(*)").
		From(capsTableName).
		Where(sq.And{sq.Eq{"node": node}, sq.Eq{"ver": ver}}).
		RunWith(r.conn).QueryRowContext(ctx)

	err := row.Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}

func (r *pgSQLCapabilitiesRep) FetchCapabilities(ctx context.Context, node, ver string) (*capsmodel.Capabilities, error) {
	row := sq.Select("node", "ver", "features", "form").
		From(capsTableName).
		Where(sq.And{sq.Eq{"node": node}, sq.Eq{"ver": ver}}).
		RunWith(r.conn).QueryRowContext(ctx)

	var b []byte
	var caps capsmodel.Capabilities
	err := row.Scan(&caps.Node, &caps.Ver, pq.Array(&caps.Features), &b)
	switch err {
	case nil:
		if len(b) > 0 {
			sb, err := stravaganza.NewBuilderFromBinary(b)
			if err != nil {
				return nil, err
			}
			caps.Form = sb.Build()
		}
		return &caps, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}
