/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	"errors"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/util/pool"
)

var (
	errGeneric = errors.New("pgsql: generic storage error")
)

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
