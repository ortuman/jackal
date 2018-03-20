/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package config

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	var cfg1, cfg2 Config
	b, err := ioutil.ReadFile("../testdata/config_basic.yml")
	require.Nil(t, err)
	err = FromBuffer(bytes.NewBuffer(b), &cfg1)
	require.Nil(t, err)
	FromFile("../testdata/config_basic.yml", &cfg2)
	require.Equal(t, cfg1, cfg2)
}

func TestBadConfigFile(t *testing.T) {
	var cfg Config
	err := FromFile("../testdata/not_a_config.yml", &cfg)
	require.NotNil(t, err)
}
