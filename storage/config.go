/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"errors"
	"fmt"

	"github.com/ortuman/jackal/storage/badgerdb"
	"github.com/ortuman/jackal/storage/sql"
)

const defaultMySQLPoolSize = 16

// Type represents a storage manager type.
type Type int

const (
	// MySQL represents a MySQL storage type.
	MySQL Type = iota

	// BadgerDB represents a BadgerDB storage type.
	BadgerDB

	// Memory represents a in-memstorage storage type.
	Memory
)

var typeStringMap = map[Type]string{
	MySQL:    "MySQL",
	BadgerDB: "BadgerDB",
	Memory:   "Memory",
}

func (t Type) String() string { return typeStringMap[t] }

// Config represents an storage manager configuration.
type Config struct {
	Type     Type
	MySQL    *sql.Config
	BadgerDB *badgerdb.Config
}

type storageProxyType struct {
	Type     string           `yaml:"type"`
	MySQL    *sql.Config      `yaml:"mysql"`
	BadgerDB *badgerdb.Config `yaml:"badgerdb"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := storageProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	switch p.Type {
	case "mysql":
		if p.MySQL == nil {
			return errors.New("storage.Config: couldn't read MySQL configuration")
		}
		c.Type = MySQL

		// assign storage defaults
		c.MySQL = p.MySQL
		if c.MySQL != nil && c.MySQL.PoolSize == 0 {
			c.MySQL.PoolSize = defaultMySQLPoolSize
		}

	case "badgerdb":
		if p.BadgerDB == nil {
			return errors.New("storage.Config: couldn't read BadgerDB configuration")
		}
		c.Type = BadgerDB

		c.BadgerDB = p.BadgerDB
		if len(c.BadgerDB.DataDir) == 0 {
			c.BadgerDB.DataDir = "./data"
		}

	case "memory":
		c.Type = Memory

	case "":
		return errors.New("storage.Config: unspecified storage type")

	default:
		return fmt.Errorf("storage.Config: unrecognized storage type: %s", p.Type)
	}
	return nil
}
