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

type StorageType int

const (
	MySQL StorageType = iota
)

type Storage struct {
	Type  StorageType
	MySQL *MySQLDb
}

type MySQLDb struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	PoolSize int    `yaml:"pool_size"`
}

type storageProxyType struct {
	Type  string   `yaml:"type"`
	MySQL *MySQLDb `yaml:"mysql"`
}

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
	default:
		return fmt.Errorf("config.Storage: unrecognized storage type: %s", p.Type)
	}
	s.MySQL = p.MySQL

	// assign storage defaults
	if s.MySQL != nil && s.MySQL.PoolSize == 0 {
		s.MySQL.PoolSize = defaultMySQLPoolSize
	}
	return nil
}
