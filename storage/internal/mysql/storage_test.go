/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package mysql

import (
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/log"
)

// newMock returns a mocked MySQL storage instance.
func newStorageMock() (*mySQLStorage, sqlmock.Sqlmock) {
	db, sqlMock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return &mySQLStorage{db: db}, sqlMock
}
