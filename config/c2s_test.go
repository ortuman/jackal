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

func TestC2SLoad(t *testing.T) {
	c2s := C2S{}
	err := yaml.Unmarshal([]byte("domains: [jackal.im]"), &c2s)
	require.Nil(t, err)
	require.Equal(t, "jackal.im", c2s.Domains[0])
}

func TestC2SEmptyDomains(t *testing.T) {
	c2s := C2S{}
	err := yaml.Unmarshal([]byte("domains: []"), &c2s)
	require.NotNil(t, err)
}

func TestC2SBadConfig(t *testing.T) {
	c2s := C2S{}
	err := yaml.Unmarshal([]byte("domains"), &c2s)
	require.NotNil(t, err)
}
