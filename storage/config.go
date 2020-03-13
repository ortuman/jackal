/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"errors"
	"fmt"

	"github.com/ortuman/jackal/storage/mysql"
	"github.com/ortuman/jackal/storage/pgsql"
)

// Type represents a storage manager type.
type Type int

const (
	// MySQL represents a MySQL storage type.
	MySQL Type = iota

	// PostgreSQL represents a PostgreSQL storage type.
	PostgreSQL

	// Memory represents a in-memstorage storage type.
	Memory
)

var typeStringMap = map[Type]string{
	MySQL:      "MySQL",
	PostgreSQL: "PostgreSQL",
	Memory:     "Memory",
}

func (t Type) String() string { return typeStringMap[t] }

// Config represents an storage manager configuration.
type Config struct {
	Type       Type
	MySQL      *mysql.Config
	PostgreSQL *pgsql.Config
}

type storageProxyType struct {
	Type       string        `yaml:"type"`
	MySQL      *mysql.Config `yaml:"mysql"`
	PostgreSQL *pgsql.Config `yaml:"pgsql"`
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
		c.MySQL = p.MySQL

	case "pgsql":
		if p.PostgreSQL == nil {
			return errors.New("storage.Config: couldn't read PostgreSQL configuration")
		}
		c.Type = PostgreSQL
		c.PostgreSQL = p.PostgreSQL

	case "memory":
		c.Type = Memory

	case "":
		return errors.New("storage.Config: unspecified storage type")

	default:
		return fmt.Errorf("storage.Config: unrecognized storage type: %s", p.Type)
	}

	return nil
}
