/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package storage

import (
	"sync"

	"github.com/ortuman/jackal/storage/entity"

	"github.com/ortuman/jackal/config"
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
		case config.MySQL:
			instance = newMySQLStorage()
		default:
			// should not be reached
			break
		}
	})
	return instance
}
