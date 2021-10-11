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
	"github.com/ortuman/jackal/pkg/log"
	"github.com/ortuman/jackal/pkg/storage/repository"
)

func init() {
	sq.StatementBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
}

// Config contains PgSQL configuration value.
type Config struct {
	Host            string        `fig:"host"`
	User            string        `fig:"user"`
	Password        string        `fig:"password"`
	Database        string        `fig:"database"`
	SSLMode         string        `fig:"ssl_mode" default:"disable"`
	MaxOpenConns    int           `fig:"max_open_conns"`
	MaxIdleConns    int           `fig:"max_idle_conns"`
	ConnMaxLifetime time.Duration `fig:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `fig:"conn_max_idle_time"`
}

// Repository represents a PgSQL repository implementation.
type Repository struct {
	repository.User
	repository.Last
	repository.Capabilities
	repository.Offline
	repository.BlockList
	repository.Private
	repository.Roster
	repository.VCard

	host string
	dsn  string
	cfg  Config

	db *sql.DB
}

// New creates and returns an initialized PgSQL Repository instance.
func New(cfg Config) *Repository {
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Database, cfg.SSLMode)
	return &Repository{
		host: cfg.Host,
		dsn:  dsn,
		cfg:  cfg,
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

	db.SetMaxIdleConns(r.cfg.MaxIdleConns)
	db.SetMaxOpenConns(r.cfg.MaxOpenConns)
	db.SetConnMaxIdleTime(r.cfg.ConnMaxIdleTime)
	db.SetConnMaxLifetime(r.cfg.ConnMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("pgsqlrepository: unable to verify PgSQL connection: %v", err)
	}
	log.Infow("Dialed PgSQL connection", "host", r.host)

	r.User = &pgSQLUserRep{conn: db}
	r.Last = &pgSQLLastRep{conn: db}
	r.Capabilities = &pgSQLCapabilitiesRep{conn: db}
	r.Offline = &pgSQLOfflineRep{conn: db}
	r.BlockList = &pgSQLBlockListRep{conn: db}
	r.Private = &pgSQLPrivateRep{conn: db}
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
