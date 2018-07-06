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
	require.Equal(t, time.Duration(600)*time.Second, trCfg.KeepAlive)

	rawCfg = `
bind_addr: 127.0.0.1
port: 5999
keep_alive: 200
`
	err = yaml.Unmarshal([]byte(rawCfg), &trCfg)
	require.Nil(t, err)
	require.Equal(t, "127.0.0.1", trCfg.BindAddress)
	require.Equal(t, 5999, trCfg.Port)
	require.Equal(t, time.Duration(200)*time.Second, trCfg.KeepAlive)
}

func TestConfig(t *testing.T) {
	rawCfg := `
enabled false
`
	cfg := Config{}
	err := yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.NotNil(t, err)

	rawCfg = `
enabled: false
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.Nil(t, err)

	rawCfg = `
enabled: true
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.NotNil(t, err) // missing dialback secret

	rawCfg = `
enabled: true
dialback_secret: s3cr3t
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.Nil(t, err) // defaults
	require.Equal(t, defaultDialTimeout, cfg.DialTimeout)
	require.Equal(t, defaultConnectTimeout, cfg.ConnectTimeout)
	require.Equal(t, defaultMaxStanzaSize, cfg.MaxStanzaSize)

	rawCfg = `
enabled: true
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
