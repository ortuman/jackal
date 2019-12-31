/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // SQL driver
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/pool"
)

// Storage represents a SQL storage sub system.
type Storage struct {
	db     *sql.DB
	pool   *pool.BufferPool
	doneCh chan chan bool
}

// New instantiates a SQL storage instance.
func New2(cfg *Config) *Storage {
	var err error
	s := &Storage{
		pool:   pool.NewBufferPool(),
		doneCh: make(chan chan bool),
	}
	host := cfg.Host
	user := cfg.User
	pass := cfg.Password
	db := cfg.Database
	poolSize := cfg.PoolSize

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, pass, host, db)
	s.db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("%v", err)
	}
	s.db.SetMaxOpenConns(poolSize) // set max opened connection count

	if err := s.db.Ping(); err != nil {
		log.Fatalf("%v", err)
	}
	go s.loop()

	return s
}

// IsClusterCompatible returns whether or not the underlying storage subsystem can be used in cluster mode.
func (s *Storage) IsClusterCompatible() bool { return true }

// Close shuts down SQL storage sub system.
func (s *Storage) Close() error {
	ch := make(chan bool)
	s.doneCh <- ch
	<-ch
	return nil
}

func (s *Storage) loop() {
	tc := time.NewTicker(time.Second * 15)
	defer tc.Stop()
	for {
		select {
		case <-tc.C:
			err := s.db.Ping()
			if err != nil {
				log.Error(err)
			}
		case ch := <-s.doneCh:
			_ = s.db.Close()
			close(ch)
			return
		}
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
