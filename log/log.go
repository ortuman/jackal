/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
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

// Logger object is used to log messages for a specific system or application component.
type Logger struct {
	level       config.LogLevel
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
	if instance().level > config.DebugLevel {
		return
	}
	ci := getCallerInfo()
	instance().debugf(ci.filename, ci.line, format, args...)
}

// Infof logs an 'info' message to the log file
// and echoes it to the console.
func Infof(format string, args ...interface{}) {
	if instance().level > config.InfoLevel {
		return
	}
	ci := getCallerInfo()
	instance().infof(ci.filename, ci.line, format, args...)
}

// Warnf logs a 'warning' message to the log file
// and echoes it to the console.
func Warnf(format string, args ...interface{}) {
	if instance().level > config.WarningLevel {
		return
	}
	ci := getCallerInfo()
	instance().warnf(ci.filename, ci.line, format, args...)
}

// Errorf logs an 'error' message to the log file
// and echoes it to the console.
func Errorf(format string, args ...interface{}) {
	if instance().level > config.ErrorLevel {
		return
	}
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

// Error logs an 'error' value
func Error(err error) {
	Errorf("%v", err)
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
	level      config.LogLevel
	file       string
	line       int
	log        string
	continueCh chan struct{}
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
	l.level = config.DefaultConfig.Logger.Level
	logFile := config.DefaultConfig.Logger.LogFile

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
	l.writeLog(file, line, format, config.DebugLevel, true, args...)
}

func (l *Logger) infof(file string, line int, format string, args ...interface{}) {
	l.writeLog(file, line, format, config.InfoLevel, true, args...)
}

func (l *Logger) warnf(file string, line int, format string, args ...interface{}) {
	l.writeLog(file, line, format, config.WarningLevel, true, args...)
}

func (l *Logger) errorf(file string, line int, format string, args ...interface{}) {
	l.writeLog(file, line, format, config.ErrorLevel, true, args...)
}

func (l *Logger) fatalf(file string, line int, format string, args ...interface{}) {
	l.writeLog(file, line, format, config.FatalLevel, false, args...)
}

func (l *Logger) writeLog(file string, line int, format string, logLevel config.LogLevel, async bool, args ...interface{}) {
	if !l.initialized {
		return
	}
	entry := record{
		level:      logLevel,
		file:       file,
		line:       line,
		log:        fmt.Sprintf(format, args...),
		continueCh: make(chan struct{}),
	}
	l.logChan <- entry
	if !async {
		<-entry.continueCh // wait until done
	}
}

func (l *Logger) loop() {
	for {
		rec := <-l.logChan

		t := time.Now()
		tm := t.Format("2006-01-02 15:04:05")

		glyph := logLevelGlyph(rec.level)
		abbr := logLevelAbbreviation(rec.level)

		line := fmt.Sprintf("%s %s [%s] %s:%d - %s\n", tm, glyph, abbr, rec.file, rec.line, rec.log)

		fmt.Print(line)
		l.f.WriteString(line)

		if rec.level == config.FatalLevel {
			os.Exit(1)
		}
		close(rec.continueCh)
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

func logLevelAbbreviation(logLevel config.LogLevel) string {
	switch logLevel {
	case config.DebugLevel:
		return "DBG"
	case config.InfoLevel:
		return "INF"
	case config.WarningLevel:
		return "WRN"
	case config.ErrorLevel:
		return "ERR"
	case config.FatalLevel:
		return "FTL"
	default:
		// should not be reached
		return ""
	}
}

func logLevelGlyph(logLevel config.LogLevel) string {
	switch logLevel {
	case config.DebugLevel:
		return "\U0001f50D"
	case config.InfoLevel:
		return "\u2139\ufe0f"
	case config.WarningLevel:
		return "\u26a0\ufe0f"
	case config.ErrorLevel:
		return "\U0001f4a5"
	case config.FatalLevel:
		return "\U0001f480"
	default:
		// should not be reached
		return ""
	}
}
