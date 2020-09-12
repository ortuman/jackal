/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const cfgExample = `
host: conference.localhost
name: "Test Server"
`

func TestXEP0045_MucConfig(t *testing.T) {
	badCfg := `host:`
	cfg := &Config{}
	err := yaml.Unmarshal([]byte(badCfg), &cfg)
	require.NotNil(t, err)

	goodCfg := cfgExample
	cfg = &Config{}
	err = yaml.Unmarshal([]byte(goodCfg), &cfg)
	require.Nil(t, err)
	require.Equal(t, cfg.MucHost, "conference.localhost")
	require.Equal(t, cfg.Name, "Test Server")
	require.NotNil(t, cfg.RoomDefaults)
}
