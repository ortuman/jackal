/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package sql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	sq "github.com/Masterminds/squirrel"
	_ "github.com/go-sql-driver/mysql" // SQL driver
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
	doneCh chan chan bool
}

// New returns a SQL storage instance.
func New(cfg *Config) *Storage {
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

// NewMock returns a mocked SQL storage instance.
func NewMock() (*Storage, sqlmock.Sqlmock) {
	var err error
	var sqlMock sqlmock.Sqlmock
	s := &Storage{
		pool: pool.NewBufferPool(),
	}
	s.db, sqlMock, err = sqlmock.New()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return s, sqlMock
}

// Shutdown shuts down SQL storage sub system.
func (s *Storage) Shutdown() {
	ch := make(chan bool)
	s.doneCh <- ch
	<-ch
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
