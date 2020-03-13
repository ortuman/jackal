/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"testing"

	"github.com/ortuman/jackal/storage/mysql"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestStorageConfig(t *testing.T) {
	cfg := Config{}

	memCfg := `
  type: memory
`
	err := yaml.Unmarshal([]byte(memCfg), &cfg)
	require.Nil(t, err)
	require.Equal(t, Memory, cfg.Type)

	mySQLCfg := `
  type: mysql
  mysql:
    host: 127.0.0.1
    user: jackal
    password: password
    database: jackaldb
    pool_size: 16
`

	err = yaml.Unmarshal([]byte(mySQLCfg), &cfg)
	require.Nil(t, err)
	require.Equal(t, MySQL, cfg.Type)
	require.Equal(t, "jackal", cfg.MySQL.User)
	require.Equal(t, "password", cfg.MySQL.Password)
	require.Equal(t, "jackaldb", cfg.MySQL.Database)
	require.Equal(t, 16, cfg.MySQL.PoolSize)

	mySQLCfg2 := `
  type: mysql
  mysql:
    host: 127.0.0.1
    user: jackal
    password: password
    database: jackaldb
`

	err = yaml.Unmarshal([]byte(mySQLCfg2), &cfg)
	require.Nil(t, err)
	require.Equal(t, MySQL, cfg.Type)
	require.Equal(t, mysql.DefaultPoolSize, cfg.MySQL.PoolSize)

	invalidMySQLCfg := `
  type: mysql
`
	err = yaml.Unmarshal([]byte(invalidMySQLCfg), &cfg)
	require.NotNil(t, err)

	invalidCfg := `
  type: invalid
`
	err = yaml.Unmarshal([]byte(invalidCfg), &cfg)
	require.NotNil(t, err)
}

func TestStorageBadConfig(t *testing.T) {
	cfg := Config{}

	memCfg := `
  type
`
	err := yaml.Unmarshal([]byte(memCfg), &cfg)
	require.NotNil(t, err)
}
