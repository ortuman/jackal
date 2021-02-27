// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pgsqlrepository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/repository"
)

func init() {
	sq.StatementBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
}

// Options contais PgSQL configuration options.
type Options struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

// Repository represents a PgSQL repository implementation.
type Repository struct {
	repository.User
	repository.Capabilities
	repository.Offline
	repository.BlockList
	repository.Roster
	repository.VCard

	host string
	dsn  string
	opts Options

	db *sql.DB
}

// New creates and returns an initialized PgSQL Repository instance.
func New(host, username, password, database, sslMode string, opts Options) *Repository {
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", username, password, host, database, sslMode)
	return &Repository{
		host: host,
		dsn:  dsn,
		opts: opts,
	}
}

// InTransaction generates a PgSQL transaction and completes it after it's being used by f function.
func (r *Repository) InTransaction(ctx context.Context, f func(ctx context.Context, tx repository.Transaction) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	repTx := newRepTx(tx)
	if err := f(ctx, repTx); err != nil {
		if err := tx.Rollback(); err != nil {
			log.Warnf("Failed to rollback PgSQL transaction: %v", err)
		}
		return err
	}
	return tx.Commit()
}

// Start implements Start interface method.
func (r *Repository) Start(ctx context.Context) error {
	db, err := sql.Open("postgres", r.dsn)
	if err != nil {
		return fmt.Errorf("pgsqlrepository: failed to start PgSQL connection: %v", err)
	}
	r.db = db

	db.SetMaxIdleConns(r.opts.MaxIdleConns)
	db.SetMaxOpenConns(r.opts.MaxOpenConns)
	db.SetConnMaxIdleTime(r.opts.ConnMaxIdleTime)
	db.SetConnMaxLifetime(r.opts.ConnMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("pgsqlrepository: unable to verify PgSQL connection: %v", err)
	}
	log.Infow("Dialed PgSQL connection", "host", r.host)

	r.User = &pgSQLUserRep{conn: db}
	r.Capabilities = &pgSQLCapabilitiesRep{conn: db}
	r.Offline = &pgSQLOfflineRep{conn: db}
	r.BlockList = &pgSQLBlockListRep{conn: db}
	r.Roster = &pgSQLRosterRep{conn: db}
	r.VCard = &pgSQLVCardRep{conn: db}
	return nil
}

// Stop closes PgSQL database and prevents new queries from starting.
func (r *Repository) Stop(_ context.Context) error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("pgsqlrepository: failed to close PgSQL connection: %v", err)
	}
	log.Infow("Closed PgSQL connection", "host", r.host)
	return nil
}

func closeRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		log.Warnf("Failed to close SQL rows: %v", err)
	}
}
