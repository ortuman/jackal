/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"os"
	"testing"

	"github.com/ortuman/jackal/transport"
	"github.com/ortuman/jackal/transport/compress"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
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

func TestTransportConfig(t *testing.T) {
	s := TransportConfig{}

	err := yaml.Unmarshal([]byte("{type: socket, bind_addr: 0.0.0.0, port: 5222, keep_alive: 120}"), &s)
	require.Nil(t, err)

	require.Equal(t, transport.Socket, s.Type)
	require.Equal(t, "0.0.0.0", s.BindAddress)
	require.Equal(t, 5222, s.Port)
}

func TestConfig(t *testing.T) {
	defer os.RemoveAll("./.cert")

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
sasl: [plain, digest_md5, scram_sha_1, scram_sha_256, scram_sha_512]
`
	err = yaml.Unmarshal([]byte(authCfg), &s)
	require.Nil(t, err)
	require.Equal(t, 5, len(s.SASL))

	// invalid auth mechanism...
	err = yaml.Unmarshal([]byte("{id: default, type: c2s, sasl: [invalid]}"), &s)
	require.NotNil(t, err)

	// invalid yaml
	err = yaml.Unmarshal([]byte("type"), &s)
	require.NotNil(t, err)
}
