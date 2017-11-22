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
	"time"

	"github.com/ortuman/jackal/config"
)

const (
	debugLevel = iota
	infoLevel
	warningLevel
	errorLevel
	fatalLevel
)

// singleton interface
var (
	loggerInst *Logger
	once       sync.Once
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

func instance() *Logger {
	once.Do(func() {
		loggerInst = &Logger{}
	})
	return loggerInst
}

func Initialize() error {
	return instance().initialize()
}

func Debugf(format string, args ...interface{}) {
	instance().writeLog(format, debugLevel, args...)
}

func Infof(format string, args ...interface{}) {
	instance().writeLog(format, infoLevel, args...)
}

func Warnf(format string, args ...interface{}) {
	instance().writeLog(format, warningLevel, args...)
}

func Errorf(format string, args ...interface{}) {
	instance().writeLog(format, errorLevel, args...)
}

func Fatalf(format string, args ...interface{}) {
	instance().writeLog(format, fatalLevel, args...)
}

func (l *Logger) initialize() error {
	if l.initialized {
		return nil
	}
	logLevel := config.DefaultConfig.Logger.Level
	logFile := config.DefaultConfig.Logger.LogFile

	switch strings.ToLower(logLevel) {
	case "debug":
		l.level = debugLevel
	case "info":
		l.level = infoLevel
	case "warning":
		l.level = warningLevel
	case "error":
		l.level = errorLevel
	case "fatal":
		l.level = fatalLevel
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

		t := time.Now()
		line := fmt.Sprintf("%s %s [%s] - %s", t.Format("2006-01-02 15:04:05"), logLevelGlyph(entry.level), logLevelAbbreviation(entry.level), entry.log)

		if entry.level >= l.level {
			fmt.Fprint(os.Stdout, line)
		}
	}
}

func (l *Logger) writeLog(format string, logLevel int, args ...interface{}) {
	entry := logEntry{
		level: logLevel,
		log:   fmt.Sprintf(format, args...),
	}
	l.logChan <- entry
}

func logLevelAbbreviation(logLevel int) string {
	switch logLevel {
	case debugLevel:
		return "DBG"
	case infoLevel:
		return "INF"
	case warningLevel:
		return "WRN"
	case errorLevel:
		return "ERR"
	case fatalLevel:
		return "FTL"
	default:
		// should not be reached
		return ""
	}
}

func logLevelGlyph(logLevel int) string {
	switch logLevel {
	case debugLevel:
		return "\U0001f50D"
	case infoLevel:
		return "\u2139\ufe0f"
	case warningLevel:
		return "\u26a0\ufe0f"
	case errorLevel:
		return "\U0001f4a5"
	case fatalLevel:
		return "\U0001f480"
	default:
		// should not be reached
		return ""
	}
}
