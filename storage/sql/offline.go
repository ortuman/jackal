/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/xml"
)

// InsertOfflineMessage inserts a new message element into
// user's offline queue.
func (s *Storage) InsertOfflineMessage(message xml.XElement, username string) error {
	q := sq.Insert("offline_messages").
		Columns("username", "data", "created_at").
		Values(username, message.String(), nowExpr)
	_, err := q.RunWith(s.db).Exec()
	return err
}

// CountOfflineMessages returns current length of user's offline queue.
func (s *Storage) CountOfflineMessages(username string) (int, error) {
	q := sq.Select("COUNT(*)").
		From("offline_messages").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at")

	var count int
	err := q.RunWith(s.db).Scan(&count)
	switch err {
	case nil:
		return count, nil
	default:
		return 0, err
	}
}

// FetchOfflineMessages retrieves from storage current user offline queue.
func (s *Storage) FetchOfflineMessages(username string) ([]xml.XElement, error) {
	q := sq.Select("data").
		From("offline_messages").
		Where(sq.Eq{"username": username}).
		OrderBy("created_at")

	rows, err := q.RunWith(s.db).Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buf := s.pool.Get()
	defer s.pool.Put(buf)

	buf.WriteString("<root>")
	for rows.Next() {
		var msg string
		rows.Scan(&msg)
		buf.WriteString(msg)
	}
	buf.WriteString("</root>")

	parser := xml.NewParser(buf, xml.DefaultMode, 0)
	rootEl, err := parser.ParseElement()
	if err != nil {
		return nil, err
	}
	return rootEl.Elements().All(), nil
}

// DeleteOfflineMessages clears a user offline queue.
func (s *Storage) DeleteOfflineMessages(username string) error {
	q := sq.Delete("offline_messages").Where(sq.Eq{"username": username})
	_, err := q.RunWith(s.db).Exec()
	return err
}
