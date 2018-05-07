/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestC2SLoad(t *testing.T) {
	cfg := Config{}
	err := yaml.Unmarshal([]byte("domains: [jackal.im]"), &cfg)
	require.Nil(t, err)
	require.Equal(t, "jackal.im", cfg.Domains[0])
}

func TestC2SEmptyDomains(t *testing.T) {
	cfg := Config{}
	err := yaml.Unmarshal([]byte("domains: []"), &cfg)
	require.NotNil(t, err)
}

func TestC2SBadConfig(t *testing.T) {
	cfg := Config{}
	err := yaml.Unmarshal([]byte("domains"), &cfg)
	require.NotNil(t, err)
}
