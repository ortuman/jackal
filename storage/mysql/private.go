/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp"
)

type mySQLPrivate struct {
	*mySQLStorage
	pool *pool.BufferPool
}

func newPrivate(db *sql.DB) *mySQLPrivate {
	return &mySQLPrivate{
		mySQLStorage: newStorage(db),
		pool:         pool.NewBufferPool(),
	}
}

// UpsertPrivateXML inserts a new private element into storage,
// or updates it in case it's been previously inserted.
func (s *mySQLPrivate) UpsertPrivateXML(ctx context.Context, privateXML []xmpp.XElement, namespace string, username string) error {
	buf := s.pool.Get()
	defer s.pool.Put(buf)
	for _, elem := range privateXML {
		if err := elem.ToXML(buf, true); err != nil {
			return err
		}
	}
	rawXML := buf.String()

	q := sq.Insert("private_storage").
		Columns("username", "namespace", "data", "updated_at", "created_at").
		Values(username, namespace, rawXML, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE data = ?, updated_at = NOW()", rawXML)

	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}

// FetchPrivateXML retrieves from storage a private element.
func (s *mySQLPrivate) FetchPrivateXML(ctx context.Context, namespace string, username string) ([]xmpp.XElement, error) {
	q := sq.Select("data").
		From("private_storage").
		Where(sq.And{sq.Eq{"username": username}, sq.Eq{"namespace": namespace}})

	var privateXML string
	err := q.RunWith(s.db).QueryRowContext(ctx).Scan(&privateXML)
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
