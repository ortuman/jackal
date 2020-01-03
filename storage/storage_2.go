/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"fmt"
	"sync"

	"github.com/ortuman/jackal/storage/badgerdb"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/mysql"
	"github.com/ortuman/jackal/storage/pgsql"
)

// Storage represents an entity storage interface.
type Storage interface {
	Close() error

	IsClusterCompatible() bool

	capabilitiesStorage
	offlineStorage
	rosterStorage
	blockListStorage
	pubSubStorage
}

var (
	instMu sync.RWMutex
	inst   Storage
)

// Disabled stores a disabled storage instance.
var Disabled Storage = &disabledStorage{}

func init() {
	inst = Disabled
}

// New2 initializes storage sub system.
func New2(config *Config) (Storage, error) {
	switch config.Type {
	case BadgerDB:
		return badgerdb.New2(config.BadgerDB), nil
	case MySQL:
		return mysql.New2(config.MySQL), nil
	case PostgreSQL:
		return pgsql.New2(config.PostgreSQL), nil
	case Memory:
		return memorystorage.New2(), nil
	default:
		return nil, fmt.Errorf("storage: unrecognized storage type: %d", config.Type)
	}
}

// Set sets the global storage.
func Set(storage Storage) {
	instMu.Lock()
	_ = inst.Close()
	inst = storage
	instMu.Unlock()
}

// Unset disables a previously set global storage.
func Unset() {
	Set(Disabled)
}

// IsClusterCompatible returns whether or not the underlying storage subsystem can be used in cluster mode.
func IsClusterCompatible() bool {
	return instance().IsClusterCompatible()
}

// instance returns a singleton instance of the storage subsystem
func instance() Storage {
	instMu.RLock()
	s := inst
	instMu.RUnlock()
	return s
}
