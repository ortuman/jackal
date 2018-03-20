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

func TestLoggerConfig(t *testing.T) {
	l := Logger{}
	err := yaml.Unmarshal([]byte("{level: debug}"), &l)
	require.Nil(t, err)
	require.Equal(t, DebugLevel, l.Level)

	err = yaml.Unmarshal([]byte("{level: info}"), &l)
	require.Nil(t, err)
	require.Equal(t, InfoLevel, l.Level)

	err = yaml.Unmarshal([]byte("{level: warning}"), &l)
	require.Nil(t, err)
	require.Equal(t, WarningLevel, l.Level)

	err = yaml.Unmarshal([]byte("{level: error}"), &l)
	require.Nil(t, err)
	require.Equal(t, ErrorLevel, l.Level)

	err = yaml.Unmarshal([]byte("{level: fatal}"), &l)
	require.Nil(t, err)
	require.Equal(t, FatalLevel, l.Level)

	err = yaml.Unmarshal([]byte("{level: invalid}"), &l)
	require.NotNil(t, err)

	err = yaml.Unmarshal([]byte("{log_path: jackal.log}"), &l)
	require.Nil(t, err)
	require.Equal(t, "jackal.log", l.LogPath)
}

func TestLoggerBadConfig(t *testing.T) {
	l := Logger{}
	err := yaml.Unmarshal([]byte("level"), &l)
	require.NotNil(t, err)
}
