/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"errors"
	"fmt"
)

const defaultMySQLPoolSize = 16

// StorageType represents a storage manager type.
type StorageType int

const (
	// MySQL represents a MySQL storage type.
	MySQL StorageType = iota

	// BadgerDB represents a BadgerDB storage type.
	BadgerDB

	// Mock represents a in-memory storage type.
	Mock
)

// Config represents an storage manager configuration.
type Config struct {
	Type     StorageType
	MySQL    *MySQLDb
	BadgerDB *BadgerDb
}

// MySQLDb represents MySQL storage configuration.
type MySQLDb struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	PoolSize int    `yaml:"pool_size"`
}

// BadgerDb represents BadgerDB storage configuration.
type BadgerDb struct {
	DataDir string `yaml:"data_dir"`
}

type storageProxyType struct {
	Type     string    `yaml:"type"`
	MySQL    *MySQLDb  `yaml:"mysql"`
	BadgerDB *BadgerDb `yaml:"badgerdb"`
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

	case "mock":
		c.Type = Mock

	case "":
		return errors.New("storage.Config: unspecified storage type")

	default:
		return fmt.Errorf("storage.Config: unrecognized storage type: %s", p.Type)
	}
	return nil
}
