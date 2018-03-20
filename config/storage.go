/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package config

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

// Storage represents an storage manager configuration.
type Storage struct {
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
func (s *Storage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	p := storageProxyType{}
	if err := unmarshal(&p); err != nil {
		return err
	}
	switch p.Type {
	case "mysql":
		if p.MySQL == nil {
			return errors.New("config.Storage: couldn't read MySQL configuration")
		}
		s.Type = MySQL

		// assign storage defaults
		s.MySQL = p.MySQL
		if s.MySQL != nil && s.MySQL.PoolSize == 0 {
			s.MySQL.PoolSize = defaultMySQLPoolSize
		}

	case "badgerdb":
		if p.BadgerDB == nil {
			return errors.New("config.Storage: couldn't read BadgerDB configuration")
		}
		s.Type = BadgerDB

		s.BadgerDB = p.BadgerDB
		if len(s.BadgerDB.DataDir) == 0 {
			s.BadgerDB.DataDir = "./data"
		}

	case "mock":
		s.Type = Mock

	case "":
		return errors.New("config.Storage: unspecified storage type")

	default:
		return fmt.Errorf("config.Storage: unrecognized storage type: %s", p.Type)
	}
	return nil
}
