/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/ortuman/jackal/storage/model"
)

func (s *Storage) InsertOrUpdateUser(u *model.User) error {
	q := sq.Insert("users").
		Columns("username", "password", "logged_out_status", "logged_out_at", "updated_at", "created_at").
		Values(u.Username, u.Password, u.LoggedOutStatus, nowExpr, nowExpr, nowExpr).
		Suffix("ON DUPLICATE KEY UPDATE password = ?, logged_out_status = ?, logged_out_at = ?, updated_at = NOW()", u.Password, u.LoggedOutStatus, u.LoggedOutAt)

	_, err := q.RunWith(s.db).Exec()
	return err
}

func (s *Storage) FetchUser(username string) (*model.User, error) {
	q := sq.Select("username", "password", "logged_out_status", "logged_out_at").
		From("users").
		Where(sq.Eq{"username": username})

	var usr model.User
	err := q.RunWith(s.db).QueryRow().Scan(&usr.Username, &usr.Password, &usr.LoggedOutStatus, &usr.LoggedOutAt)
	switch err {
	case nil:
		return &usr, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

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
