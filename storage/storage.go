/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"fmt"

	"github.com/ortuman/jackal/storage/internal/badgerdb"
	"github.com/ortuman/jackal/storage/internal/memory"
	"github.com/ortuman/jackal/storage/internal/mysql"
	"github.com/ortuman/jackal/storage/internal/pgsql"
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
		return memory.New()
	default:
		return nil, fmt.Errorf("storage: unrecognized storage type: %d", config.Type)
	}
}
