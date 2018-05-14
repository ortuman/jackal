/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/xml"
)

func (s *Storage) InsertOrUpdateVCard(vCard xml.XElement, username string) error {
	rawXML := vCard.String()
	q := sq.Insert("vcards").
		Columns("username", "vcard", "updated_at", "created_at").
		Values(username, rawXML, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE vcard = ?, updated_at = NOW()", rawXML)

	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *Storage) FetchVCard(username string) (xml.XElement, error) {
	q := sq.Select("vcard").From("vcards").Where(sq.Eq{"username": username})

	var vCard string
	err := q.RunWith(s.db).QueryRow().Scan(&vCard)
	switch err {
	case nil:
		parser := xml.NewParser(strings.NewReader(vCard))
		return parser.ParseElement()
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}
