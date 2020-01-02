/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"fmt"

	"github.com/ortuman/jackal/storage/badgerdb"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/mysql"
	"github.com/ortuman/jackal/storage/pgsql"
	"github.com/ortuman/jackal/storage/repository"
)

func New(config *Config) (repository.Container, error) {
	switch config.Type {
	case BadgerDB:
		return badgerdb.New(config.BadgerDB)
	case MySQL:
		return mysql.New(config.MySQL)
	case PostgreSQL:
		return pgsql.New(config.PostgreSQL)
	case Memory:
		return memorystorage.New()
	default:
		return nil, fmt.Errorf("storage: unrecognized storage type: %d", config.Type)
	}
}
