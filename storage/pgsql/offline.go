/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type pgSQLOffline struct {
	*pgSQLStorage
	pool *pool.BufferPool
}

func newOffline(db *sql.DB) *pgSQLOffline {
	return &pgSQLOffline{
		pgSQLStorage: newStorage(db),
		pool:         pool.NewBufferPool(),
	}
}

// InsertOfflineMessage inserts a new message element into user's offline queue.
func (s *pgSQLOffline) InsertOfflineMessage(ctx context.Context, message *xmpp.Message, username string) error {
	q := sq.Insert("offline_messages").
		Columns("username", "data").
		Values(username, message.String())

	_, err := q.RunWith(s.db).ExecContext(ctx)

	return err
}

// CountOfflineMessages returns current length of user's offline queue.
func (s *pgSQLOffline) CountOfflineMessages(ctx context.Context, username string) (int, error) {
	var count int

	q := sq.Select("COUNT(*)").
		From("offline_messages").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at")

	if err := q.RunWith(s.db).QueryRowContext(ctx).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// FetchOfflineMessages retrieves from storage current user offline queue.
func (s *pgSQLOffline) FetchOfflineMessages(ctx context.Context, username string) ([]xmpp.Message, error) {
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

// DeleteOfflineMessages clears a user offline queue.
func (s *pgSQLOffline) DeleteOfflineMessages(ctx context.Context, username string) error {
	q := sq.Delete("offline_messages").Where(sq.Eq{"username": username})
	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}
