/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// InsertOfflineMessage inserts a new message element into user's offline queue.
func (s *Storage) InsertOfflineMessage(ctx context.Context, message *xmpp.Message, username string) error {
	q := sq.Insert("offline_messages").
		Columns("username", "data", "created_at").
		Values(username, message.String(), nowExpr)
	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}

// CountOfflineMessages returns current length of user's offline queue.
func (s *Storage) CountOfflineMessages(ctx context.Context, username string) (int, error) {
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

// FetchOfflineMessages retrieves from storage current user offline queue.
func (s *Storage) FetchOfflineMessages(ctx context.Context, username string) ([]xmpp.Message, error) {
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
	elems := rootEl.Elements().All()

	var messages []xmpp.Message
	for _, el := range elems {
		fromJID, _ := jid.NewWithString(el.From(), true)
		toJID, _ := jid.NewWithString(el.To(), true)
		msg, err := xmpp.NewMessageFromElement(el, fromJID, toJID)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *msg)
	}
	return messages, nil
}

// DeleteOfflineMessages clears a user offline queue.
func (s *Storage) DeleteOfflineMessages(ctx context.Context, username string) error {
	q := sq.Delete("offline_messages").Where(sq.Eq{"username": username})
	_, err := q.RunWith(s.db).ExecContext(ctx)
	return err
}
