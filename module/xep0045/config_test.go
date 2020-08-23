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

func TestMucConfig(t *testing.T) {
	badCfg := `service:`
	cfg := &Config{}
	err := yaml.Unmarshal([]byte(badCfg), &cfg)
	require.NotNil(t, err)

	goodCfg := `service: conference.jackal.im`
	cfg = &Config{}
	err = yaml.Unmarshal([]byte(goodCfg), &cfg)
	require.Nil(t, err)
}
