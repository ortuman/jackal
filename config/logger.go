/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package config

import (
	"fmt"
	"strings"
)

// LogLevel represents log level type.
type LogLevel int

const (
	// DebugLevel represents DEBUG log level.
	DebugLevel LogLevel = iota

	// InfoLevel represents INFO log level.
	InfoLevel

	// WarningLevel represents WARNING log level.
	WarningLevel

	// ErrorLevel represents ERROR log level.
	ErrorLevel

	// FatalLevel represents FATAL log level.
	FatalLevel
)

// Logger represents a logger manager configuration.
type Logger struct {
	Level   LogLevel
	LogPath string
}

type loggerProxyType struct {
	Level   string `yaml:"level"`
	LogPath string `yaml:"log_path"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (l *Logger) UnmarshalYAML(unmarshal func(interface{}) error) error {
	lp := loggerProxyType{}
	if err := unmarshal(&lp); err != nil {
		return err
	}
	switch strings.ToLower(lp.Level) {
	case "debug":
		l.Level = DebugLevel
	case "", "info": // default log level
		l.Level = InfoLevel
	case "warning":
		l.Level = WarningLevel
	case "error":
		l.Level = ErrorLevel
	case "fatal":
		l.Level = FatalLevel
	default:
		return fmt.Errorf("config.Logger: unrecognized log level: %s", lp.Level)
	}
	l.LogPath = lp.LogPath
	return nil
}
