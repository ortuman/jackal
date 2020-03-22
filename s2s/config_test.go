/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestTransportConfig(t *testing.T) {
	rawCfg := `
bind_addr 0.0.0.0
`
	trCfg := TransportConfig{}
	err := yaml.Unmarshal([]byte(rawCfg), &trCfg)
	require.NotNil(t, err)

	rawCfg = `
bind_addr: 0.0.0.0
`
	err = yaml.Unmarshal([]byte(rawCfg), &trCfg)
	require.Nil(t, err)
	require.Equal(t, "0.0.0.0", trCfg.BindAddress)
	require.Equal(t, 5269, trCfg.Port)

	rawCfg = `
bind_addr: 127.0.0.1
port: 5999
`
	err = yaml.Unmarshal([]byte(rawCfg), &trCfg)
	require.Nil(t, err)
	require.Equal(t, "127.0.0.1", trCfg.BindAddress)
	require.Equal(t, 5999, trCfg.Port)
}

func TestConfig(t *testing.T) {
	cfg := Config{}
	rawCfg := `
dial_timeout: 300
connect_timeout: 250
max_stanza_size: 8192
`
	err := yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.NotNil(t, err) // missing dialback secret

	rawCfg = `
dialback_secret: s3cr3t
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.Nil(t, err) // defaults
	require.Equal(t, defaultDialTimeout, cfg.DialTimeout)
	require.Equal(t, defaultConnectTimeout, cfg.ConnectTimeout)
	require.Equal(t, defaultMaxStanzaSize, cfg.MaxStanzaSize)

	rawCfg = `
dialback_secret: s3cr3t
dial_timeout: 300
connect_timeout: 250
max_stanza_size: 8192
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.Nil(t, err) // defaults
	require.Equal(t, time.Duration(300)*time.Second, cfg.DialTimeout)
	require.Equal(t, time.Duration(250)*time.Second, cfg.ConnectTimeout)
	require.Equal(t, 8192, cfg.MaxStanzaSize)
}
