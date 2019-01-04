/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/xmpp"
)

// InsertOrUpdatePrivateXML inserts a new private element into storage,
// or updates it in case it's been previously inserted.
func (s *Storage) InsertOrUpdatePrivateXML(privateXML []xmpp.XElement, namespace string, username string) error {
	buf := s.pool.Get()
	defer s.pool.Put(buf)
	for _, elem := range privateXML {
		elem.ToXML(buf, true)
	}
	rawXML := buf.String()

	var suffix string

	switch s.engine {
	case "mysql":
		suffix = "ON DUPLICATE KEY UPDATE data = ?, updated_at = NOW()"
	case "postgresql":
		suffix = "ON CONFLICT (username) DO UPDATE SET data = $4, updated_at = NOW()"
	}

	q := sq.Insert("private_storage").
		Columns("username", "namespace", "data", "updated_at", "created_at").
		Values(username, namespace, rawXML, nowExpr, nowExpr).
		Suffix(suffix, rawXML)

	_, err := q.RunWith(s.db).Exec()
	return err
}

// FetchPrivateXML retrieves from storage a private element.
func (s *Storage) FetchPrivateXML(namespace string, username string) ([]xmpp.XElement, error) {
	q := sq.Select("data").
		From("private_storage").
		Where(sq.And{sq.Eq{"username": username}, sq.Eq{"namespace": namespace}})

	var privateXML string
	err := q.RunWith(s.db).QueryRow().Scan(&privateXML)
	switch err {
	case nil:
		buf := s.pool.Get()
		defer s.pool.Put(buf)
		buf.WriteString("<root>")
		buf.WriteString(privateXML)
		buf.WriteString("</root>")

		parser := xmpp.NewParser(buf, xmpp.DefaultMode, 0)
		rootEl, err := parser.ParseElement()
		if err != nil {
			return nil, err
		}
		return rootEl.Elements().All(), nil

	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}
