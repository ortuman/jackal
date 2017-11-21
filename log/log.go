/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ortuman/jackal/config"
)

const (
	debug = iota
	info
	warning
	error
	fatal
)

// singleton interface
var (
	instance *Logger
	once     sync.Once
)

type logEntry struct {
	level int
	log   string
}

type Logger struct {
	level       int
	f           *os.File
	logChan     chan logEntry
	initialized bool
}

func Instance() *Logger {
	once.Do(func() {
		instance = &Logger{}
	})
	return instance
}

func (l *Logger) Initialize() error {
	if l.initialized {
		return nil
	}
	logLevel := config.DefaultConfig.Logger.Level
	logFile := config.DefaultConfig.Logger.LogFile

	switch strings.ToLower(logLevel) {
	case "debug":
		l.level = debug
	case "info":
		l.level = info
	case "warning":
		l.level = warning
	case "error":
		l.level = error
	case "fatal":
		l.level = fatal
	default:
		return fmt.Errorf("unrecognized log level: %s", logLevel)
	}

	// create logFile intermediate directories.
	if err := os.MkdirAll(filepath.Dir(logFile), os.ModePerm); err != nil {
		return err
	}
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	l.f = f
	l.logChan = make(chan logEntry, 256)

	go l.loop()

	l.initialized = true
	return nil
}

func (l *Logger) loop() {
	for {
		entry := <-l.logChan
		if entry.level >= l.level {

		}
	}
}
