/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestStorageConfig(t *testing.T) {
	s := Storage{}

	mockCfg := `
  type: mock
`
	err := yaml.Unmarshal([]byte(mockCfg), &s)
	require.Nil(t, err)
	require.Equal(t, Mock, s.Type)

	mySQLCfg := `
  type: mysql
  mysql:
    host: 127.0.0.1
    user: jackal
    password: password
    database: jackaldb
    pool_size: 16
`

	err = yaml.Unmarshal([]byte(mySQLCfg), &s)
	require.Nil(t, err)
	require.Equal(t, MySQL, s.Type)
	require.Equal(t, "jackal", s.MySQL.User)
	require.Equal(t, "password", s.MySQL.Password)
	require.Equal(t, "jackaldb", s.MySQL.Database)
	require.Equal(t, 16, s.MySQL.PoolSize)

	mySQLCfg2 := `
  type: mysql
  mysql:
    host: 127.0.0.1
    user: jackal
    password: password
    database: jackaldb
`

	err = yaml.Unmarshal([]byte(mySQLCfg2), &s)
	require.Nil(t, err)
	require.Equal(t, MySQL, s.Type)
	require.Equal(t, defaultMySQLPoolSize, s.MySQL.PoolSize)

	invalidMySQLCfg := `
  type: mysql
`
	err = yaml.Unmarshal([]byte(invalidMySQLCfg), &s)
	require.NotNil(t, err)

	invalidCfg := `
  type: invalid
`
	err = yaml.Unmarshal([]byte(invalidCfg), &s)
	require.NotNil(t, err)
}

func TestStorageBadConfig(t *testing.T) {
	s := Storage{}

	mockCfg := `
  type
`
	err := yaml.Unmarshal([]byte(mockCfg), &s)
	require.NotNil(t, err)
}
