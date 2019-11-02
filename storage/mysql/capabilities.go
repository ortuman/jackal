package mysql

import (
	"database/sql"
	"encoding/json"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

func (s *Storage) InsertCapabilities(node, ver string, features []string) error {
	b, err := json.Marshal(&features)
	if err != nil {
		return err
	}
	_, err = sq.Insert("capabilities").
		Columns("node", "ver", "features", "created_at").
		Values(node, ver, b, nowExpr).
		RunWith(s.db).Exec()
	return err
}

func (s *Storage) HasCapabilities(node, ver string) (bool, error) {
	var count int
	err := sq.Select("COUNT(*)").From("capabilities").
		Where(sq.And{sq.Eq{"node": node}, sq.Eq{"ver": ver}}).
		RunWith(s.db).QueryRow().Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}

func (s *Storage) FetchCapabilities(node, ver string) ([]string, error) {
	var b string
	err := sq.Select("features").From("capabilities").
		Where(sq.And{sq.Eq{"node": node}, sq.Eq{"ver": ver}}).
		RunWith(s.db).QueryRow().Scan(&b)
	switch err {
	case nil:
		var features []string
		if err := json.NewDecoder(strings.NewReader(b)).Decode(&features); err != nil {
			return nil, err
		}
		return features, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}
