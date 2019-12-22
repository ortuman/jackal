/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"database/sql"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

// UpsertUser inserts a new user entity into storage,
// or updates it in case it's been previously inserted.
func (s *Storage) UpsertUser(u *model.User) error {
	var presenceXML string

	if u.LastPresence != nil {
		buf := s.pool.Get()
		u.LastPresence.ToXML(buf, true)
		presenceXML = buf.String()
		s.pool.Put(buf)
	}

	q := sq.Insert("users")

	if len(presenceXML) > 0 {
		q = q.Columns("username", "password", "last_presence", "last_presence_at").
			Values(u.Username, u.Password, presenceXML, nowExpr).
			Suffix("ON CONFLICT (username) DO UPDATE SET password = $2, last_presence = $3, last_presence_at = NOW()")
	} else {
		q = q.Columns("username", "password").
			Values(u.Username, u.Password).
			Suffix("ON CONFLICT (username) DO UPDATE SET password = $2")
	}
	_, err := q.RunWith(s.db).Exec()
	return err
}

// FetchUser retrieves from storage a user entity.
func (s *Storage) FetchUser(username string) (*model.User, error) {
	q := sq.Select("username", "password", "last_presence", "last_presence_at").
		From("users").
		Where(sq.Eq{"username": username})

	var presenceXML string
	var presenceAt time.Time
	var usr model.User

	err := q.RunWith(s.db).QueryRow().Scan(&usr.Username, &usr.Password, &presenceXML, &presenceAt)
	switch err {
	case nil:
		if len(presenceXML) > 0 {
			parser := xmpp.NewParser(strings.NewReader(presenceXML), xmpp.DefaultMode, 0)
			lastPresence, err := parser.ParseElement()
			if err != nil {
				return nil, err
			}
			fromJID, _ := jid.NewWithString(lastPresence.From(), true)
			toJID, _ := jid.NewWithString(lastPresence.To(), true)
			usr.LastPresence, _ = xmpp.NewPresenceFromElement(lastPresence, fromJID, toJID)
			usr.LastPresenceAt = presenceAt
		}
		return &usr, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

// DeleteUser deletes a user entity from storage.
func (s *Storage) DeleteUser(username string) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		var err error
		_, err = sq.Delete("offline_messages").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		_, err = sq.Delete("roster_items").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		_, err = sq.Delete("roster_versions").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		_, err = sq.Delete("private_storage").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		_, err = sq.Delete("vcards").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		_, err = sq.Delete("users").Where(sq.Eq{"username": username}).RunWith(tx).Exec()
		if err != nil {
			return err
		}
		return nil
	})
}

// UserExists returns whether or not a user exists within storage.
func (s *Storage) UserExists(username string) (bool, error) {
	q := sq.Select("COUNT(*)").From("users").Where(sq.Eq{"username": username})
	var count int
	err := q.RunWith(s.db).QueryRow().Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}
