/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestModuleConfig(t *testing.T) {
	badCfg := `enabled [roster]`
	cfg := &Config{}
	err := yaml.Unmarshal([]byte(badCfg), &cfg)
	require.NotNil(t, err)
	badMod := `enabled: [bad_mod]`
	err = yaml.Unmarshal([]byte(badMod), &cfg)
	require.NotNil(t, err)
	validMod := `enabled: [roster]`
	err = yaml.Unmarshal([]byte(validMod), &cfg)
	require.Nil(t, err)
}
