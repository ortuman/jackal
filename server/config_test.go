/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package server

import (
	"testing"

	"github.com/ortuman/jackal/server/transport"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestTypeStrings(t *testing.T) {
	require.Equal(t, "c2s", C2SServerType.String())
	require.Equal(t, "s2s", S2SServerType.String())
	require.Equal(t, "", ServerType(99).String())
}

func TestConfig(t *testing.T) {
	s := Config{}

	// s2s not yet supported...
	err := yaml.Unmarshal([]byte("{id: default, type: s2s}"), &s)
	require.NotNil(t, err)

	err = yaml.Unmarshal([]byte("{id: default, type: c2s}"), &s)
	require.Nil(t, err)
	require.Equal(t, "default", s.ID)
	require.Equal(t, C2SServerType, s.Type)
}

func TestTlS(t *testing.T) {
	s := TLSConfig{}

	err := yaml.Unmarshal([]byte("{privkey_path: key.pem, cert_path: cert.pem}"), &s)
	require.Nil(t, err)
	require.Equal(t, "key.pem", s.PrivKeyFile)
	require.Equal(t, "cert.pem", s.CertFile)
}

func TestTransportConfig(t *testing.T) {
	cfg := `
type: socket
bind_addr: 192.168.0.1
port: 6666
keep_alive: 240
`
	tr := TransportConfig{}
	err := yaml.Unmarshal([]byte(cfg), &tr)
	require.Nil(t, err)
	require.Equal(t, transport.Socket, tr.Type)
	require.Equal(t, "192.168.0.1", tr.BindAddress)
	require.Equal(t, 6666, tr.Port)
	require.Equal(t, 240, tr.KeepAlive)

	// test defaults
	err = yaml.Unmarshal([]byte("{type: socket}"), &tr)
	require.Nil(t, err)
	require.Equal(t, transport.Socket, tr.Type)
	require.Equal(t, "", tr.BindAddress)
	require.Equal(t, defaultTransportPort, tr.Port)
	require.Equal(t, defaultTransportKeepAlive, tr.KeepAlive)

	// invalid transport type
	err = yaml.Unmarshal([]byte("{type: invalid}"), &tr)
	require.NotNil(t, err)

	// invalid yaml
	err = yaml.Unmarshal([]byte("type"), &tr)
	require.NotNil(t, err)
}
