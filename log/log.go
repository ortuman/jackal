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

// Logger object is used to log messages for a specific system or application component
type Logger struct {
	tag string
}

// NewLogger creates a new Logger instance with a tag log prefix.
func NewLogger(tag string) *Logger {
	l := &Logger{tag: tag}
	return l
}

// Debugf logs a [DEBUG] message to the log file.
// Also echoes the message to the console.
func (l *Logger) Debugf(format string, args ...interface{}) {
	instance().writeLog(fmt.Sprintf("%s: %s", l.tag, format), debugLevel, args...)
}

// Infof logs a [INFO] message to the log file.
// Also echoes the message to the console.
func (l *Logger) Infof(format string, args ...interface{}) {
	instance().writeLog(fmt.Sprintf("%s: %s", l.tag, format), infoLevel, args...)
}

// Warnf logs a [WARN] message to the log file.
// Also echoes the message to the console.
func (l *Logger) Warnf(format string, args ...interface{}) {
	instance().writeLog(fmt.Sprintf("%s: %s", l.tag, format), warningLevel, args...)
}

// Errorf logs a [ERROR] message to the log file.
// Also echoes the message to the console.
func (l *Logger) Errorf(format string, args ...interface{}) {
	instance().writeLog(fmt.Sprintf("%s: %s", l.tag, format), errorLevel, args...)
}

// Fatalf logs a [FATAL] message to the log file.
// Also echoes the message to the console.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	instance().writeLog(fmt.Sprintf("%s: %s", l.tag, format), fatalLevel, args...)
}

// singleton interface
var (
	logInst *log
	once    sync.Once
)

type logEntry struct {
	level int
	log   string
}

type log struct {
	level       int
	f           *os.File
	logChan     chan logEntry
	initialized bool
}

func instance() *log {
	once.Do(func() {
		logInst = &log{}
	})
	return logInst
}

// Initialize initalizes logger subsystem.
func Initialize() error {
	logLevel := config.DefaultConfig.Logger.Level
	logFile := config.DefaultConfig.Logger.LogFile
	return instance().initialize(logLevel, logFile)
}

func (l *log) initialize(logLevel, logFile string) error {
	if l.initialized {
		return nil
	}

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

func (l *log) writeLog(format string, logLevel int, args ...interface{}) {
	if !l.initialized {
		return
	}
	entry := logEntry{
		level: logLevel,
		log:   fmt.Sprintf(format, args...),
	}
	l.logChan <- entry
}

func (l *log) loop() {
	for {
		entry := <-l.logChan

		t := time.Now()
		line := fmt.Sprintf("%s %s [%s] - %s\n", t.Format("2006-01-02 15:04:05"), logLevelGlyph(entry.level), logLevelAbbreviation(entry.level), entry.log)

		if entry.level >= l.level {
			fmt.Fprint(os.Stdout, line)
			l.f.WriteString(line)
		}
		if entry.level == fatalLevel {
			os.Exit(1)
		}
	}
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
