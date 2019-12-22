package mysql

import (
	"database/sql"
	"encoding/json"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/model"
)

func (s *Storage) InsertCapabilities(caps *model.Capabilities) error {
	b, err := json.Marshal(caps.Features)
	if err != nil {
		return err
	}
	_, err = sq.Insert("capabilities").
		Columns("node", "ver", "features", "created_at").
		Values(caps.Node, caps.Ver, b, nowExpr).
		RunWith(s.db).Exec()
	return err
}

func (s *Storage) FetchCapabilities(node, ver string) (*model.Capabilities, error) {
	var b string
	err := sq.Select("features").From("capabilities").
		Where(sq.And{sq.Eq{"node": node}, sq.Eq{"ver": ver}}).
		RunWith(s.db).QueryRow().Scan(&b)
	switch err {
	case nil:
		var caps model.Capabilities
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
