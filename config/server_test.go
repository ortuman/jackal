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

func TestTypeStrings(t *testing.T) {
	require.Equal(t, "c2s", C2SServerType.String())
	require.Equal(t, "s2s", S2SServerType.String())
	require.Equal(t, "", ServerType(99).String())

	require.Equal(t, "socket", SocketTransportType.String())
	require.Equal(t, "", TransportType(99).String())

	require.Equal(t, "default", DefaultCompression.String())
	require.Equal(t, "best", BestCompression.String())
	require.Equal(t, "speed", SpeedCompression.String())
	require.Equal(t, "", CompressionLevel(99).String())
}

func TestCompressionConfig(t *testing.T) {
	cmp := Compression{}
	err := yaml.Unmarshal([]byte("{level: default}"), &cmp)
	require.Nil(t, err)
	require.Equal(t, DefaultCompression, cmp.Level)

	err = yaml.Unmarshal([]byte("{level: best}"), &cmp)
	require.Nil(t, err)
	require.Equal(t, BestCompression, cmp.Level)

	err = yaml.Unmarshal([]byte("{level: speed}"), &cmp)
	require.Nil(t, err)
	require.Equal(t, SpeedCompression, cmp.Level)

	err = yaml.Unmarshal([]byte("{level: unknown}"), &cmp)
	require.NotNil(t, err)

	err = yaml.Unmarshal([]byte("level"), &cmp)
	require.NotNil(t, err)
}

func TestTransportConfig(t *testing.T) {
	cfg := `
type: socket
bind_addr: 192.168.0.1
port: 6666
connect_timeout: 10
keep_alive: 240
max_stanza_size: 8192
`
	tr := Transport{}
	err := yaml.Unmarshal([]byte(cfg), &tr)
	require.Nil(t, err)
	require.Equal(t, SocketTransportType, tr.Type)
	require.Equal(t, "192.168.0.1", tr.BindAddress)
	require.Equal(t, 6666, tr.Port)
	require.Equal(t, 10, tr.ConnectTimeout)
	require.Equal(t, 240, tr.KeepAlive)
	require.Equal(t, 8192, tr.MaxStanzaSize)

	// test defaults
	err = yaml.Unmarshal([]byte("{type: socket}"), &tr)
	require.Nil(t, err)
	require.Equal(t, SocketTransportType, tr.Type)
	require.Equal(t, "", tr.BindAddress)
	require.Equal(t, defaultTransportPort, tr.Port)
	require.Equal(t, defaultTransportConnectTimeout, tr.ConnectTimeout)
	require.Equal(t, defaultTransportKeepAlive, tr.KeepAlive)
	require.Equal(t, defaultTransportMaxStanzaSize, tr.MaxStanzaSize)

	// invalid transport type
	err = yaml.Unmarshal([]byte("{type: invalid}"), &tr)
	require.NotNil(t, err)

	// invalid yaml
	err = yaml.Unmarshal([]byte("type"), &tr)
	require.NotNil(t, err)
}

func TestServerConfig(t *testing.T) {
	s := Server{}
	err := yaml.Unmarshal([]byte("{id: default, type: c2s}"), &s)
	require.Nil(t, err)

	// s2s not yet supported...
	err = yaml.Unmarshal([]byte("{id: default, type: s2s}"), &s)
	require.NotNil(t, err)

	// resource conflict options...
	err = yaml.Unmarshal([]byte("{id: default, type: c2s, resource_conflict: reject}"), &s)
	require.Nil(t, err)

	err = yaml.Unmarshal([]byte("{id: default, type: c2s, resource_conflict: override}"), &s)
	require.Nil(t, err)

	// invalid resource conflict option...
	err = yaml.Unmarshal([]byte("{id: default, type: c2s, resource_conflict: invalid}"), &s)
	require.NotNil(t, err)

	// auth mechanisms...
	authCfg := `
id: default
type: c2s
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
id: default
type: c2s
modules: [roster, private, vcard, registration, version, ping, offline]
`
	err = yaml.Unmarshal([]byte(modulesCfg), &s)
	require.Nil(t, err)

	// invalid server module...
	err = yaml.Unmarshal([]byte("{id: default, type: c2s, modules: [invalid]}"), &s)
	require.NotNil(t, err)

	// invalid type
	err = yaml.Unmarshal([]byte("{id: default, type: invalid}"), &s)
	require.NotNil(t, err)

	// invalid yaml
	err = yaml.Unmarshal([]byte("type"), &s)
	require.NotNil(t, err)
}
