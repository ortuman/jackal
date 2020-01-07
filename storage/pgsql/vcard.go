/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/xmpp"
)

type pgSQLVCard struct {
	*pgSQLStorage
}

func newVCard(db *sql.DB) *pgSQLVCard {
	return &pgSQLVCard{
		pgSQLStorage: newStorage(db),
	}
}

// UpsertVCard inserts a new vCard element into storage, or updates it in case it's been previously inserted.
func (s *pgSQLVCard) UpsertVCard(ctx context.Context, vCard xmpp.XElement, username string) error {
	rawXML := vCard.String()

	q := sq.Insert("vcards").
		Columns("username", "vcard").
		Values(username, rawXML).
		Suffix("ON CONFLICT (username) DO UPDATE SET vcard = $3", rawXML)

	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}

// FetchVCard retrieves from storage a vCard element associated to a given user.
func (s *pgSQLVCard) FetchVCard(ctx context.Context, username string) (xmpp.XElement, error) {
	q := sq.Select("vcard").From("vcards").Where(sq.Eq{"username": username})

	var vCard string

	err := q.RunWith(s.db).QueryRowContext(ctx).Scan(&vCard)

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
