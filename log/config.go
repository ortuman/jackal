/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package log

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

// Config represents a logger manager configuration.
type Config struct {
	Level   LogLevel
	LogPath string
}

type configProxyType struct {
	Level   string `yaml:"level"`
	LogPath string `yaml:"log_path"`
}

// UnmarshalYAML satisfies Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	lp := configProxyType{}
	if err := unmarshal(&lp); err != nil {
		return err
	}
	switch strings.ToLower(lp.Level) {
	case "debug":
		c.Level = DebugLevel
	case "", "info": // default log level
		c.Level = InfoLevel
	case "warning":
		c.Level = WarningLevel
	case "error":
		c.Level = ErrorLevel
	case "fatal":
		c.Level = FatalLevel
	default:
		return fmt.Errorf("log.Config: unrecognized log level: %s", lp.Level)
	}
	c.LogPath = lp.LogPath
	return nil
}
