// Copyright 2022 The jackal Authors
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
	"github.com/cockroachdb/errors"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/ortuman/jackal/pkg/storage/repository"
)

const noLoadBalancePrefix = "/*NO LOAD BALANCE*/"

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
	repository.Archive
	repository.Locker

	host string
	dsn  string
	cfg  Config

	db     *sql.DB
	logger kitlog.Logger
}

// New creates and returns an initialized PgSQL Repository instance.
func New(cfg Config, logger kitlog.Logger) *Repository {
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Database, cfg.SSLMode)
	return &Repository{
		host:   cfg.Host,
		dsn:    dsn,
		cfg:    cfg,
		logger: logger,
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
			level.Warn(r.logger).Log("msg", "failed to rollback PgSQL transaction", "err", err)
		}
		return err
	}
	return tx.Commit()
}

// Start implements Start interface method.
func (r *Repository) Start(ctx context.Context) error {
	db, err := sql.Open("postgres", r.dsn)
	if err != nil {
		return errors.Wrap(err, "failed to start PgSQL connection")
	}
	r.db = db

	db.SetMaxIdleConns(r.cfg.MaxIdleConns)
	db.SetMaxOpenConns(r.cfg.MaxOpenConns)
	db.SetConnMaxIdleTime(r.cfg.ConnMaxIdleTime)
	db.SetConnMaxLifetime(r.cfg.ConnMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		return errors.Wrap(err, "unable to verify PgSQL connection")
	}
	level.Info(r.logger).Log("msg", "dialed PgSQL connection", "host", r.host)

	r.User = &pgSQLUserRep{conn: db, logger: r.logger}
	r.Last = &pgSQLLastRep{conn: db, logger: r.logger}
	r.Capabilities = &pgSQLCapabilitiesRep{conn: db, logger: r.logger}
	r.Offline = &pgSQLOfflineRep{conn: db, logger: r.logger}
	r.BlockList = &pgSQLBlockListRep{conn: db, logger: r.logger}
	r.Private = &pgSQLPrivateRep{conn: db, logger: r.logger}
	r.Roster = &pgSQLRosterRep{conn: db, logger: r.logger}
	r.VCard = &pgSQLVCardRep{conn: db, logger: r.logger}
	r.Archive = &pgSQLArchiveRep{conn: db, logger: r.logger}
	r.Locker = &pgSQLLocker{conn: db}
	return nil
}

// Stop closes PgSQL database and prevents new queries from starting.
func (r *Repository) Stop(_ context.Context) error {
	if err := r.db.Close(); err != nil {
		return errors.Wrap(err, "failed to close PgSQL connection")
	}
	level.Info(r.logger).Log("msg", "closed PgSQL connection", "host", r.host)
	return nil
}

func closeRows(rows *sql.Rows, logger kitlog.Logger) {
	if err := rows.Close(); err != nil {
		level.Warn(logger).Log("msg", "failed to close SQL rows", "err", err)
	}
}
