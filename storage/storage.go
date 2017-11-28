/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package storage

import (
	"sync"

	"github.com/ortuman/jackal/storage/entity"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/log"
)

type storage interface {
	// User
	FetchUser(username string) (*entity.User, error)

	InsertOrUpdate(user entity.User) error
	DeleteUser(username string) error
}

// singleton interface
var (
	instance storage
	once     sync.Once
)

func Instance() storage {
	once.Do(func() {
		switch config.DefaultConfig.Storage.Type {
		case "mysql":
			instance = newMySQLStorage()
		default:
			log.Fatalf("unrecognized storage type: %s", config.DefaultConfig.Storage.Type)
			return
		}
	})
	return instance
}
