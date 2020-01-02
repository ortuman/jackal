/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pgsql

import (
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/log"
)

// newStorageMock returns a mocked MySQL storage instance.
func newStorageMock() (*pgSQLStorage, sqlmock.Sqlmock) {
	db, sqlMock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return &pgSQLStorage{db: db}, sqlMock
}
