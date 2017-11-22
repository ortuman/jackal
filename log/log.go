/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package log

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

// Logger object is used to log messages for a specific system or application component.
type Logger struct {
	level       int
	f           *os.File
	logChan     chan record
	initialized bool
}

// Initialize initalizes logger subsystem.
func Initialize() error {
	return instance().initialize()
}

// Debugf logs a 'debug' message to the log file
// and echoes it to the console.
func Debugf(format string, args ...interface{}) {
	ci := getCallerInfo()
	instance().debugf(ci.filename, ci.line, format, args...)
}

// Infof logs an 'info' message to the log file
// and echoes it to the console.
func Infof(format string, args ...interface{}) {
	ci := getCallerInfo()
	instance().infof(ci.filename, ci.line, format, args...)
}

// Warnf logs a 'warning' message to the log file
// and echoes it to the console.
func Warnf(format string, args ...interface{}) {
	ci := getCallerInfo()
	instance().warnf(ci.filename, ci.line, format, args...)
}

// Errorf logs an 'error' message to the log file
// and echoes it to the console.
func Errorf(format string, args ...interface{}) {
	ci := getCallerInfo()
	instance().errorf(ci.filename, ci.line, format, args...)
}

// Fatalf logs a 'fatal' message to the log file
// and echoes it to the console.
// Application will terminate after logging.
func Fatalf(format string, args ...interface{}) {
	ci := getCallerInfo()
	instance().fatalf(ci.filename, ci.line, format, args...)
}

// singleton interface
var (
	logInst *Logger
	once    sync.Once
)

type callerInfo struct {
	filename string
	line     int
}

type record struct {
	level int
	file  string
	line  int
	log   string
}

func instance() *Logger {
	once.Do(func() {
		logInst = &Logger{}
	})
	return logInst
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
	l.logChan = make(chan record, 256)

	go l.loop()

	l.initialized = true
	return nil
}

func (l *Logger) debugf(file string, line int, format string, args ...interface{}) {
	l.writeLog(file, line, format, debugLevel, args...)
}

func (l *Logger) infof(file string, line int, format string, args ...interface{}) {
	l.writeLog(file, line, format, infoLevel, args...)
}

func (l *Logger) warnf(file string, line int, format string, args ...interface{}) {
	l.writeLog(file, line, format, warningLevel, args...)
}

func (l *Logger) errorf(file string, line int, format string, args ...interface{}) {
	l.writeLog(file, line, format, errorLevel, args...)
}

func (l *Logger) fatalf(file string, line int, format string, args ...interface{}) {
	l.writeLog(file, line, format, fatalLevel, args...)
}

func (l *Logger) writeLog(file string, line int, format string, logLevel int, args ...interface{}) {
	if !l.initialized {
		return
	}
	entry := record{
		level: logLevel,
		file:  file,
		line:  line,
		log:   fmt.Sprintf(format, args...),
	}
	l.logChan <- entry
}

func (l *Logger) loop() {
	for {
		rec := <-l.logChan

		t := time.Now()
		tm := t.Format("2006-01-02 15:04:05")

		glyph := logLevelGlyph(rec.level)
		abbr := logLevelAbbreviation(rec.level)

		line := fmt.Sprintf("%s %s [%s] %s:%d - %s\n", tm, glyph, abbr, rec.file, rec.line, rec.log)

		if rec.level >= l.level {
			fmt.Fprint(os.Stdout, line)
			l.f.WriteString(line)
		}
		if rec.level == fatalLevel {
			os.Exit(1)
		}
	}
}

func getCallerInfo() callerInfo {
	_, file, ln, ok := runtime.Caller(2)
	if !ok {
		file = "???"
	}
	ci := callerInfo{}
	filename := filepath.Base(file)
	ci.filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	ci.line = ln
	return ci
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
