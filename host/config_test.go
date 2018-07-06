/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package host

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestConfig(t *testing.T) {
	rawCfg := `
name jackal.im`
	cfg := Config{}
	err := yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.NotNil(t, err)

	rawCfg = `
name: localhost
tls:
  privkey_path: "../testdata/cert/test.server.key"
  cert_path: "../testdata/cert/test.server.crt"`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.Nil(t, err)
}
