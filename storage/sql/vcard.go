/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/xmpp"
)

// InsertOrUpdateVCard inserts a new vCard element into storage,
// or updates it in case it's been previously inserted.
func (s *Storage) InsertOrUpdateVCard(vCard xmpp.XElement, username string) error {
	var suffix string

	rawXML := vCard.String()

	switch s.engine {
	case "mysql":
		suffix = "ON DUPLICATE KEY UPDATE vcard = ?, updated_at = NOW()"
	case "postgresql":
		suffix = "ON CONFLICT (username) DO UPDATE SET vcard = $3, updated_at = NOW()"
	}

	q := sq.Insert("vcards").
		Columns("username", "vcard", "updated_at", "created_at").
		Values(username, rawXML, nowExpr, nowExpr).
		Suffix(suffix, rawXML)

	_, err := q.RunWith(s.db).Exec()
	return err
}

// FetchVCard retrieves from storage a vCard element associated
// to a given user.
func (s *Storage) FetchVCard(username string) (xmpp.XElement, error) {
	q := sq.Select("vcard").From("vcards").Where(sq.Eq{"username": username})

	var vCard string
	err := q.RunWith(s.db).QueryRow().Scan(&vCard)
	switch err {
	case nil:
		parser := xmpp.NewParser(strings.NewReader(vCard), xmpp.DefaultMode, 0)
		return parser.ParseElement()
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}
