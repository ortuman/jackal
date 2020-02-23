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
	"github.com/ortuman/jackal/xmpp/jid"
)

type mySQLOffline struct {
	*mySQLStorage
	pool *pool.BufferPool
}

func newOffline(db *sql.DB) *mySQLOffline {
	return &mySQLOffline{
		mySQLStorage: newStorage(db),
		pool:         pool.NewBufferPool(),
	}
}

func (s *mySQLOffline) InsertOfflineMessage(ctx context.Context, message *xmpp.Message, username string) error {
	q := sq.Insert("offline_messages").
		Columns("username", "data", "created_at").
		Values(username, message.String(), nowExpr)
	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *mySQLOffline) CountOfflineMessages(ctx context.Context, username string) (int, error) {
	q := sq.Select("COUNT(*)").
		From("offline_messages").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at")

	var count int
	err := q.RunWith(s.db).QueryRowContext(ctx).Scan(&count)
	switch err {
	case nil:
		return count, nil
	default:
		return 0, err
	}
}

func (s *mySQLOffline) FetchOfflineMessages(ctx context.Context, username string) ([]xmpp.Message, error) {
	q := sq.Select("data").
		From("offline_messages").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at")

	rows, err := q.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	buf := s.pool.Get()
	defer s.pool.Put(buf)

	buf.WriteString("<r>")
	for rows.Next() {
		var msg string
		if err := rows.Scan(&msg); err != nil {
			return nil, err
		}
		buf.WriteString(msg)
	}
	buf.WriteString("</r>")

	parser := xmpp.NewParser(buf, xmpp.DefaultMode, 0)
	rootEl, err := parser.ParseElement()
	if err != nil {
		return nil, err
	}
	elements := rootEl.Elements().All()

	messages := make([]xmpp.Message, len(elements))
	for i, el := range elements {
		fromJID, _ := jid.NewWithString(el.From(), true)
		toJID, _ := jid.NewWithString(el.To(), true)

		msg, err := xmpp.NewMessageFromElement(el, fromJID, toJID)
		if err != nil {
			return nil, err
		}
		messages[i] = *msg
	}
	return messages, nil
}

func (s *mySQLOffline) DeleteOfflineMessages(ctx context.Context, username string) error {
	q := sq.Delete("offline_messages").Where(sq.Eq{"username": username})
	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}
