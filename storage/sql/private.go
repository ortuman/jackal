/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/xml"
)

func (s *Storage) InsertOrUpdatePrivateXML(privateXML []xml.XElement, namespace string, username string) error {
	buf := s.pool.Get()
	defer s.pool.Put(buf)
	for _, elem := range privateXML {
		elem.ToXML(buf, true)
	}
	rawXML := buf.String()

	q := sq.Insert("private_storage").
		Columns("username", "namespace", "data", "updated_at", "created_at").
		Values(username, namespace, rawXML, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE data = ?, updated_at = NOW()", rawXML)

	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *Storage) FetchPrivateXML(namespace string, username string) ([]xml.XElement, error) {
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

		parser := xml.NewParser(buf)
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
