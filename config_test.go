/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	var cfg1, cfg2 Config
	b, err := ioutil.ReadFile("./testdata/config_basic.yml")
	require.Nil(t, err)
	err = cfg1.FromBuffer(bytes.NewBuffer(b))
	require.Nil(t, err)
	cfg2.FromFile("./testdata/config_basic.yml")
	require.Equal(t, cfg1, cfg2)
}

func TestBadConfigFile(t *testing.T) {
	var cfg Config
	err := cfg.FromFile("./testdata/not_a_config.yml")
	require.NotNil(t, err)
}
