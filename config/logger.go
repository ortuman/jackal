/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package config

import (
	"fmt"
	"strings"
)

type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarningLevel
	ErrorLevel
	FatalLevel
)

type Logger struct {
	Level   LogLevel
	LogFile string
}

type loggerProxyType struct {
	Level   string `yaml:"level"`
	LogFile string `yaml:"log_path"`
}

func (l *Logger) UnmarshalYAML(unmarshal func(interface{}) error) error {
	lp := loggerProxyType{}
	if err := unmarshal(&lp); err != nil {
		return err
	}
	switch strings.ToLower(lp.Level) {
	case "debug":
		l.Level = DebugLevel
	case "info":
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
	l.LogFile = lp.LogFile
	return nil
}
