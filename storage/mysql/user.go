/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/util/pool"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
)

type mySQLUser struct {
	*mySQLStorage
	pool *pool.BufferPool
}

func newUser(db *sql.DB) *mySQLUser {
	return &mySQLUser{
		mySQLStorage: newStorage(db),
		pool:         pool.NewBufferPool(),
	}
}

func (u *mySQLUser) UpsertUser(ctx context.Context, usr *model.User) error {
	var presenceXML string
	if usr.LastPresence != nil {
		buf := u.pool.Get()
		if err := usr.LastPresence.ToXML(buf, true); err != nil {
			return err
		}
		presenceXML = buf.String()
		u.pool.Put(buf)
	}
	columns := []string{"username", "password_scram_sha1", "password_scram_sha256", "salt", "iteration_count", "updated_at", "created_at"}
	values := []interface{}{usr.Username, usr.PasswordScramSHA1, usr.PasswordScramSHA256, usr.Salt, usr.IterationCount, nowExpr, nowExpr}

	if len(presenceXML) > 0 {
		columns = append(columns, []string{"last_presence", "last_presence_at"}...)
		values = append(values, []interface{}{presenceXML, nowExpr}...)
	}
	var suffix string
	var suffixArgs []interface{}
	if len(presenceXML) > 0 {
		suffix = "ON DUPLICATE KEY UPDATE password_scram_sha1 = ?, password_scram_sha256 = ?, salt = ?, iteration_count = ?, last_presence = ?, last_presence_at = NOW(), updated_at = NOW()"
		suffixArgs = []interface{}{usr.PasswordScramSHA1, usr.PasswordScramSHA256, usr.Salt, usr.IterationCount, presenceXML}
	} else {
		suffix = "ON DUPLICATE KEY UPDATE password_scram_sha1 = ?, password_scram_sha256 = ?, salt = ?, updated_at = NOW()"
		suffixArgs = []interface{}{usr.PasswordScramSHA1, usr.PasswordScramSHA256, usr.Salt, usr.IterationCount}
	}
	q := sq.Insert("users").
		Columns(columns...).
		Values(values...).
		Suffix(suffix, suffixArgs...)

	_, err := q.RunWith(u.db).ExecContext(ctx)
	return err
}

func (u *mySQLUser) FetchUser(ctx context.Context, username string) (*model.User, error) {
	q := sq.Select("username", "password_scram_sha1", "password_scram_sha256", "salt", "iteration_count", "last_presence", "last_presence_at").
		From("users").
		Where(sq.Eq{"username": username})

	var presenceXML string
	var presenceAt time.Time
	var usr model.User

	err := q.RunWith(u.db).
		QueryRowContext(ctx).
		Scan(&usr.Username, &usr.PasswordScramSHA1, &usr.PasswordScramSHA256, &usr.Salt, &usr.IterationCount, &presenceXML, &presenceAt)
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

func (u *mySQLUser) DeleteUser(ctx context.Context, username string) error {
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

func (u *mySQLUser) UserExists(ctx context.Context, username string) (bool, error) {
	q := sq.Select("COUNT(*)").
		From("users").
		Where(sq.Eq{"username": username})

	var count int
	err := q.RunWith(u.db).QueryRowContext(ctx).Scan(&count)
	switch err {
	case nil:
		return count > 0, nil
	default:
		return false, err
	}
}
