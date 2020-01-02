/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
)

var (
	nowExpr = sq.Expr("NOW()")
)

type rowScanner interface {
	Scan(...interface{}) error
}

type rowsScanner interface {
	rowScanner
	Next() bool
}

// mySQLStorage represents a SQL storage sub system.
type mySQLStorage struct {
	// DB represents a MySQL database handler.
	db *sql.DB
}

var (
	errMocked = errors.New("mysql: storage error")
)

// New instantiates a MySQL base storage instance.
func newStorage(db *sql.DB) *mySQLStorage {
	return &mySQLStorage{db: db}
}

func (s *mySQLStorage) inTransaction(ctx context.Context, f func(tx *sql.Tx) error) error {
	tx, txErr := s.db.BeginTx(ctx, nil)
	if txErr != nil {
		return txErr
	}
	if err := f(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
