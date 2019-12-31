/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/pool"
)

// Storage represents a SQL storage sub system.
type Storage struct {
	db         *sql.DB
	pool       *pool.BufferPool
	cancelPing context.CancelFunc
}

// New instantiates a PostgreSQL storage instance.
func New2(c *Config) *Storage {
	var err error

	sq.StatementBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	s := &Storage{
		pool: pool.NewBufferPool(),
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", c.User, c.Password, c.Host, c.Database, c.SSLMode)

	s.db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("%v", err)
	}

	s.db.SetMaxOpenConns(c.PoolSize) // set max opened connection count

	s.ping(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelPing = cancel
	go s.pingLoop(ctx)

	return s
}

// IsClusterCompatible returns whether or not the underlying storage subsystem can be used in cluster mode.
func (s *Storage) IsClusterCompatible() bool { return true }

// Close shuts down SQL storage sub system.
func (s *Storage) Close() error {
	s.cancelPing() // Stop pinging the server

	return s.db.Close()
}

// pingLoop periodically pings the server to check connection status
func (s *Storage) pingLoop(ctx context.Context) {
	tick := time.NewTicker(pingInterval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			s.ping(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// ping sends a ping request to the server and outputs any error to log
func (s *Storage) ping(ctx context.Context) {
	pingCtx, cancel := context.WithDeadline(ctx, time.Now().Add(pingTimeout))
	defer cancel()

	err := s.db.PingContext(pingCtx)

	if err != nil {
		log.Error(err)
	}
}

func (s *Storage) inTransaction(ctx context.Context, f func(tx *sql.Tx) error) error {
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
