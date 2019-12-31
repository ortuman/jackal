/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/pool"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type user struct {
	*pgSQLStorage
	pool *pool.BufferPool
}

func newUser(db *sql.DB) *user {
	return &user{
		pgSQLStorage: newStorage(db),
		pool:         pool.NewBufferPool(),
	}
}

// UpsertUser inserts a new user entity into storage, or updates it in case it's been previously inserted.
func (u *user) UpsertUser(ctx context.Context, usr *model.User) error {
	var presenceXML string

	if usr.LastPresence != nil {
		buf := u.pool.Get()
		if err := usr.LastPresence.ToXML(buf, true); err != nil {
			return err
		}
		presenceXML = buf.String()
		u.pool.Put(buf)
	}

	q := sq.Insert("users")

	if len(presenceXML) > 0 {
		q = q.Columns("username", "password", "last_presence", "last_presence_at").
			Values(usr.Username, usr.Password, presenceXML, nowExpr).
			Suffix("ON CONFLICT (username) DO UPDATE SET password = $2, last_presence = $3, last_presence_at = NOW()")
	} else {
		q = q.Columns("username", "password").
			Values(usr.Username, usr.Password).
			Suffix("ON CONFLICT (username) DO UPDATE SET password = $2")
	}
	_, err := q.RunWith(u.db).ExecContext(ctx)
	return err
}

// FetchUser retrieves from storage a user entity.
func (u *user) FetchUser(ctx context.Context, username string) (*model.User, error) {
	q := sq.Select("username", "password", "last_presence", "last_presence_at").
		From("users").
		Where(sq.Eq{"username": username})

	var presenceXML string
	var presenceAt time.Time
	var usr model.User

	err := q.RunWith(u.db).QueryRowContext(ctx).Scan(&usr.Username, &usr.Password, &presenceXML, &presenceAt)
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
func (u *user) DeleteUser(ctx context.Context, username string) error {
	return u.inTransaction(ctx, func(tx *sql.Tx) error {
		var err error
		_, err = sq.Delete("offline_messages").Where(sq.Eq{"username": username}).RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		_, err = sq.Delete("roster_items").Where(sq.Eq{"username": username}).RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		_, err = sq.Delete("roster_versions").Where(sq.Eq{"username": username}).RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		_, err = sq.Delete("private_storage").Where(sq.Eq{"username": username}).RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		_, err = sq.Delete("vcards").Where(sq.Eq{"username": username}).RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		_, err = sq.Delete("users").Where(sq.Eq{"username": username}).RunWith(tx).ExecContext(ctx)
		if err != nil {
			return err
		}
		return nil
	})
}

// UserExists returns whether or not a user exists within storage.
func (u *user) UserExists(ctx context.Context, username string) (bool, error) {
	var count int

	q := sq.Select("COUNT(*)").From("users").Where(sq.Eq{"username": username})
	err := q.RunWith(u.db).QueryRowContext(ctx).Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}
