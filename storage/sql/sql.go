/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/pool"
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

// Config represents SQL storage configuration.
type Config struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	PoolSize int    `yaml:"pool_size"`
}

// Storage represents a SQL storage sub system.
type Storage struct {
	db     *sql.DB
	pool   *pool.BufferPool
	engine string
	doneCh chan chan bool
}

// New returns a SQL storage instance.
func New(engine string, cfg *Config) *Storage {
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

	var dsn, driver string

	switch engine {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, pass, host, db)
		driver = "mysql"
	case "postgresql":
		dsn = fmt.Sprintf("postgres://%s:%s@%s/%s", user, pass, host, db)
		driver = "postgres"
		sq.StatementBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	}

	s.engine = engine
	s.db, err = sql.Open(driver, dsn)

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
			s.db.Close()
			close(ch)
			return
		}
	}
}

func (s *Storage) inTransaction(f func(tx *sql.Tx) error) error {
	tx, txErr := s.db.Begin()
	if txErr != nil {
		return txErr
	}
	if err := f(tx); err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
