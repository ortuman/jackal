/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package log

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestLoggerConfig(t *testing.T) {
	c := Config{}
	err := yaml.Unmarshal([]byte("{level: debug}"), &c)
	require.Nil(t, err)
	require.Equal(t, DebugLevel, c.Level)

	err = yaml.Unmarshal([]byte("{level: info}"), &c)
	require.Nil(t, err)
	require.Equal(t, InfoLevel, c.Level)

	err = yaml.Unmarshal([]byte("{level: warning}"), &c)
	require.Nil(t, err)
	require.Equal(t, WarningLevel, c.Level)

	err = yaml.Unmarshal([]byte("{level: error}"), &c)
	require.Nil(t, err)
	require.Equal(t, ErrorLevel, c.Level)

	err = yaml.Unmarshal([]byte("{level: fatal}"), &c)
	require.Nil(t, err)
	require.Equal(t, FatalLevel, c.Level)

	err = yaml.Unmarshal([]byte("{level: invalid}"), &c)
	require.NotNil(t, err)

	err = yaml.Unmarshal([]byte("{log_path: jackal.log}"), &c)
	require.Nil(t, err)
	require.Equal(t, "jackal.log", c.LogPath)
}

func TestLoggerBadConfig(t *testing.T) {
	l := Logger{}
	err := yaml.Unmarshal([]byte("level"), &l)
	require.NotNil(t, err)
}
