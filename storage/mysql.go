/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package storage

import (
	"database/sql"
	"fmt"

	// driver implementation
	_ "github.com/go-sql-driver/mysql"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/storage/entity"
)

const defaultPoolSize = 16

const maxTransactionRetries = 4

type mySQL struct {
	db *sql.DB
}

func newMySQLStorage() storage {
	s := &mySQL{}
	host := config.DefaultConfig.Storage.MySQL.Host
	user := config.DefaultConfig.Storage.MySQL.User
	pass := config.DefaultConfig.Storage.MySQL.Password
	db := config.DefaultConfig.Storage.MySQL.Database

	poolSize := config.DefaultConfig.Storage.MySQL.PoolSize
	if poolSize == 0 {
		poolSize = defaultPoolSize
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", user, pass, host, db)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = conn.Ping()
	if err != nil {
		log.Fatalf("%v", err)
	}

	// set max opened connection count
	conn.SetMaxOpenConns(poolSize)

	s.db = conn
	return s
}

func (s *mySQL) FetchUser(username string) (*entity.User, error) {
	row := s.db.QueryRow("SELECT username, password FROM users WHERE username = ?", username)
	u := entity.User{}
	err := row.Scan(&u.Username, &u.Password)
	switch err {
	case nil:
		return &u, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (s *mySQL) InsertOrUpdate(u entity.User) error {
	stmt := `` +
		`INSERT INTO users(username, password, updated_at, created_at)` +
		`VALUES(?, ?, NOW(), NOW())` +
		`ON DUPLICATE KEY UPDATE password = ?, updated_at = NOW()`
	_, err := s.db.Exec(stmt, u.Username, u.Password, u.Password)
	return err
}

func (s *mySQL) DeleteUser(username string) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		var err error
		_, err = tx.Exec("DELETE FROM offline_messages WHERE username = ?", username)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM private_storage WHERE username = ?", username)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM vcards WHERE username = ?", username)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM users WHERE username = ?", username)
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *mySQL) inTransaction(f func(tx *sql.Tx) error) error {
	var err error
	for i := 0; i < maxTransactionRetries; i++ {
		tx, txErr := s.db.Begin()
		if txErr != nil {
			return txErr
		}
		err = f(tx)
		if err != nil {
			tx.Rollback()
			continue
		}
		tx.Commit()
	}
	return err
}
