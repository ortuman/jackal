/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

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

// pgSQLStorage represents a SQL storage base sub system.
type pgSQLStorage struct {
	db *sql.DB
}

var (
	errMocked = errors.New("pgsql: storage error")
)

// newStorage instantiates a PostgreSQL base storage instance.
func newStorage(db *sql.DB) *pgSQLStorage {
	return &pgSQLStorage{db: db}
}

func (s *pgSQLStorage) inTransaction(ctx context.Context, f func(tx *sql.Tx) error) error {
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
