/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	capsmodel "github.com/ortuman/jackal/model/capabilities"

	sq "github.com/Masterminds/squirrel"
)

type mySQLCapabilities struct {
	*mySQLStorage
}

func newCapabilities(db *sql.DB) *mySQLCapabilities {
	return &mySQLCapabilities{
		mySQLStorage: newStorage(db),
	}
}

func (s *mySQLCapabilities) UpsertCapabilities(ctx context.Context, caps *capsmodel.Capabilities) error {
	b, err := json.Marshal(caps.Features)
	if err != nil {
		return err
	}
	_, err = sq.Insert("capabilities").
		Columns("node", "ver", "features", "updated_at", "created_at").
		Values(caps.Node, caps.Ver, b, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE features = ?, updated_at = NOW()", b).
		RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *mySQLCapabilities) FetchCapabilities(ctx context.Context, node, ver string) (*capsmodel.Capabilities, error) {
	var b string
	err := sq.Select("features").From("capabilities").
		Where(sq.And{sq.Eq{"node": node}, sq.Eq{"ver": ver}}).
		RunWith(s.db).QueryRowContext(ctx).Scan(&b)
	switch err {
	case nil:
		var caps capsmodel.Capabilities
		if err := json.NewDecoder(strings.NewReader(b)).Decode(&caps.Features); err != nil {
			return nil, err
		}
		return &caps, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}
