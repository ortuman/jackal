/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"testing"

	"github.com/ortuman/jackal/server/compress"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestCompressionConfig(t *testing.T) {
	cmp := CompressConfig{}
	err := yaml.Unmarshal([]byte("{level: default}"), &cmp)
	require.Nil(t, err)
	require.Equal(t, compress.DefaultCompression, cmp.Level)

	err = yaml.Unmarshal([]byte("{level: best}"), &cmp)
	require.Nil(t, err)
	require.Equal(t, compress.BestCompression, cmp.Level)

	err = yaml.Unmarshal([]byte("{level: speed}"), &cmp)
	require.Nil(t, err)
	require.Equal(t, compress.SpeedCompression, cmp.Level)

	err = yaml.Unmarshal([]byte("{level: unknown}"), &cmp)
	require.NotNil(t, err)

	err = yaml.Unmarshal([]byte("level"), &cmp)
	require.NotNil(t, err)
}

func TestConfig(t *testing.T) {
	s := Config{}

	// resource conflict options...
	err := yaml.Unmarshal([]byte("{connect_timeout: 5, resource_conflict: reject}"), &s)
	require.Nil(t, err)

	err = yaml.Unmarshal([]byte("{connect_timeout: 5, resource_conflict: override}"), &s)
	require.Nil(t, err)

	// invalid resource conflict option...
	err = yaml.Unmarshal([]byte("{connect_timeout: 5, resource_conflict: invalid}"), &s)
	require.NotNil(t, err)

	// auth mechanisms...
	authCfg := `
connect_timeout: 5
resource_conflict: reject
sasl: [plain, digest_md5, scram_sha_1, scram_sha_256]
`
	err = yaml.Unmarshal([]byte(authCfg), &s)
	require.Nil(t, err)
	require.Equal(t, 4, len(s.SASL))

	// invalid auth mechanism...
	err = yaml.Unmarshal([]byte("{id: default, type: c2s, sasl: [invalid]}"), &s)
	require.NotNil(t, err)

	// server modules...
	modulesCfg := `
connect_timeout: 5 
resource_conflict: reject
modules:
  enabled: [roster, private, vcard, registration, version, ping, offline]
`
	err = yaml.Unmarshal([]byte(modulesCfg), &s)
	require.Nil(t, err)

	// invalid server module...
	modulesCfg = `
connect_timeout: 5 
resource_conflict: reject
modules:
  enabled: [invalid]
`
	err = yaml.Unmarshal([]byte(modulesCfg), &s)
	require.NotNil(t, err)

	// invalid yaml
	err = yaml.Unmarshal([]byte("type"), &s)
	require.NotNil(t, err)
}
