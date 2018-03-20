/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/config"
)

const logChanBufferSize = 512

var exitHandler = func() { os.Exit(-1) }

// singleton interface
var (
	inst        *Logger
	instMu      sync.RWMutex
	initialized uint32
)

// Logger object is used to log messages for a specific system or application component.
type Logger struct {
	level     config.LogLevel
	outWriter io.Writer
	errWriter io.Writer
	f         *os.File
	recCh     chan record
	closeCh   chan bool
}

func newLogger(cfg *config.Logger, outWriter io.Writer, errWriter io.Writer) (*Logger, error) {
	l := &Logger{
		level:     cfg.Level,
		outWriter: outWriter,
		errWriter: errWriter,
	}
	if len(cfg.LogPath) > 0 {
		// create logFile intermediate directories.
		if err := os.MkdirAll(filepath.Dir(cfg.LogPath), os.ModePerm); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(cfg.LogPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return nil, err
		}
		l.f = f
	}
	l.recCh = make(chan record, logChanBufferSize)
	l.closeCh = make(chan bool)
	go l.loop()
	return l, nil
}

// Initialize initializes the default log subsystem.
func Initialize(cfg *config.Logger) {
	if atomic.CompareAndSwapUint32(&initialized, 0, 1) {
		instMu.Lock()
		defer instMu.Unlock()

		l, err := newLogger(cfg, os.Stdout, os.Stderr)
		if err != nil {
			log.Fatalf("%v", err)
		}
		inst = l
	}
}

func instance() *Logger {
	instMu.RLock()
	defer instMu.RUnlock()
	return inst
}

// Shutdown shuts down log sub system.
// This method should be used only for testing purposes.
func Shutdown() {
	if atomic.CompareAndSwapUint32(&initialized, 1, 0) {
		instMu.Lock()
		defer instMu.Unlock()

		inst.closeCh <- true
		inst = nil
	}
}

// Debugf logs a 'debug' message to the log file
// and echoes it to the console.
func Debugf(format string, args ...interface{}) {
	if inst := instance(); inst != nil && inst.level <= config.DebugLevel {
		ci := getCallerInfo()
		inst.writeLog(ci.filename, ci.line, format, config.DebugLevel, true, args...)
	}
}

// Infof logs an 'info' message to the log file
// and echoes it to the console.
func Infof(format string, args ...interface{}) {
	if inst := instance(); inst != nil && inst.level <= config.InfoLevel {
		ci := getCallerInfo()
		inst.writeLog(ci.filename, ci.line, format, config.InfoLevel, true, args...)
	}
}

// Warnf logs a 'warning' message to the log file
// and echoes it to the console.
func Warnf(format string, args ...interface{}) {
	if inst := instance(); inst != nil && inst.level <= config.WarningLevel {
		ci := getCallerInfo()
		inst.writeLog(ci.filename, ci.line, format, config.WarningLevel, true, args...)
	}
}

// Errorf logs an 'error' message to the log file
// and echoes it to the console.
func Errorf(format string, args ...interface{}) {
	if inst := instance(); inst != nil && inst.level <= config.ErrorLevel {
		ci := getCallerInfo()
		inst.writeLog(ci.filename, ci.line, format, config.ErrorLevel, true, args...)
	}
}

// Error logs an 'error' value to the log file
// and echoes it to the console.
func Error(err error) {
	if inst := instance(); inst != nil && inst.level <= config.ErrorLevel {
		ci := getCallerInfo()
		inst.writeLog(ci.filename, ci.line, "%v", config.ErrorLevel, true, err)
	}
}

// Fatalf logs a 'fatal' message to the log file
// and echoes it to the console.
// Application will terminate after logging.
func Fatalf(format string, args ...interface{}) {
	if inst := instance(); inst != nil {
		ci := getCallerInfo()
		inst.writeLog(ci.filename, ci.line, format, config.FatalLevel, false, args...)
	}
}

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

func (l *Logger) writeLog(file string, line int, format string, level config.LogLevel, async bool, args ...interface{}) {
	entry := record{
		level:      level,
		file:       file,
		line:       line,
		log:        fmt.Sprintf(format, args...),
		continueCh: make(chan struct{}),
	}
	select {
	case l.recCh <- entry:
		if !async {
			<-entry.continueCh // wait until done
		}
	default:
		break // avoid blocking...
	}
}

func (l *Logger) loop() {
	for {
		select {
		case rec := <-l.recCh:
			t := time.Now()
			tm := t.Format("2006-01-02 15:04:05")

			glyph := logLevelGlyph(rec.level)
			abbr := logLevelAbbreviation(rec.level)
			line := fmt.Sprintf("%s %s [%s] %s:%d - %s\n", tm, glyph, abbr, rec.file, rec.line, rec.log)

			if l.f != nil {
				l.f.WriteString(line)
			}
			switch rec.level {
			case config.DebugLevel, config.WarningLevel, config.InfoLevel:
				fmt.Fprintf(l.outWriter, line)
			case config.ErrorLevel:
				fmt.Fprintf(l.errWriter, line)
			case config.FatalLevel:
				fmt.Fprintf(l.errWriter, line)
				exitHandler()
			}
			close(rec.continueCh)

		case <-l.closeCh:
			if l.f != nil {
				l.f.Close()
			}
			return
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
