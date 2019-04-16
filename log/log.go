/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package log

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const logChanBufferSize = 2048

const projectFolder = "jackal"

var exitHandler = func() { os.Exit(-1) }

// Level represents log level type.
type Level int

const (
	// DebugLevel represents DEBUG log level.
	DebugLevel Level = iota

	// InfoLevel represents INFO log level.
	InfoLevel

	// WarningLevel represents WARNING log level.
	WarningLevel

	// ErrorLevel represents ERROR log level.
	ErrorLevel

	// FatalLevel represents FATAL log level.
	FatalLevel

	// OffLevel represents a disabledLogger log level.
	OffLevel
)

// Logger represents a common logger interface.
type Logger interface {
	io.Closer

	Level() Level
	Log(level Level, pkg string, file string, line int, format string, args ...interface{})
}

// Debugf writes a 'debug' message to configured logger.
func Debugf(format string, args ...interface{}) {
	if inst := instance(); inst.Level() <= DebugLevel {
		ci := getCallerInfo()
		inst.Log(DebugLevel, ci.pkg, ci.filename, ci.line, format, args...)
	}
}

// Infof writes a 'info' message to configured logger.
func Infof(format string, args ...interface{}) {
	if inst := instance(); inst.Level() <= InfoLevel {
		ci := getCallerInfo()
		inst.Log(InfoLevel, ci.pkg, ci.filename, ci.line, format, args...)
	}
}

// Warnf writes a 'warning' message to configured logger.
func Warnf(format string, args ...interface{}) {
	if inst := instance(); inst.Level() <= WarningLevel {
		ci := getCallerInfo()
		inst.Log(WarningLevel, ci.pkg, ci.filename, ci.line, format, args...)
	}
}

// Errorf writes an 'error' message to configured logger.
func Errorf(format string, args ...interface{}) {
	if inst := instance(); inst.Level() <= ErrorLevel {
		ci := getCallerInfo()
		inst.Log(ErrorLevel, ci.pkg, ci.filename, ci.line, format, args...)
	}
}

// Fatalf writes a 'fatal' message to configured logger.
// Application should terminate after logging.
func Fatalf(format string, args ...interface{}) {
	if inst := instance(); inst.Level() <= FatalLevel {
		ci := getCallerInfo()
		inst.Log(FatalLevel, ci.pkg, ci.filename, ci.line, format, args...)
	}
	return
}

// Error writes an error value to configured logger.
func Error(err error) {
	if inst := instance(); inst.Level() <= ErrorLevel {
		ci := getCallerInfo()
		inst.Log(ErrorLevel, ci.pkg, ci.filename, ci.line, "%v", err)
	}
}

// Fatal writes an error value to configured logger.
// Application should terminate after logging.
func Fatal(err error) {
	if inst := instance(); inst.Level() <= FatalLevel {
		ci := getCallerInfo()
		inst.Log(FatalLevel, ci.pkg, ci.filename, ci.line, "%v", err)
	}
}

var (
	instMu sync.RWMutex
	inst   Logger
)

// Disabled stores a disabled logger instance.
var Disabled Logger = &disabledLogger{}

func init() {
	inst = Disabled
}

// Set sets the global logger.
func Set(logger Logger) {
	instMu.Lock()
	_ = inst.Close()
	inst = logger
	instMu.Unlock()
}

// Unset disables a previously set global logger.
func Unset() {
	Set(Disabled)
}

func instance() Logger {
	instMu.RLock()
	l := inst
	instMu.RUnlock()
	return l
}

type callerInfo struct {
	pkg      string
	filename string
	line     int
}

type record struct {
	level      Level
	pkg        string
	file       string
	line       int
	log        string
	continueCh chan struct{}
}

type logger struct {
	level  Level
	output io.Writer
	files  []io.WriteCloser
	b      strings.Builder
	recCh  chan record
}

// New returns a default logger instance.
func New(level string, output io.Writer, files ...io.WriteCloser) (Logger, error) {
	lvl, err := levelFromString(level)
	if err != nil {
		return nil, err
	}
	l := &logger{
		level:  lvl,
		output: output,
		files:  files,
	}
	l.recCh = make(chan record, logChanBufferSize)
	go l.loop()
	return l, nil
}

func (l *logger) Level() Level {
	return l.level
}

func (l *logger) Log(level Level, pkg string, file string, line int, format string, args ...interface{}) {
	entry := record{
		level:      level,
		pkg:        pkg,
		file:       file,
		line:       line,
		log:        fmt.Sprintf(format, args...),
		continueCh: make(chan struct{}),
	}
	select {
	case l.recCh <- entry:
		if level == FatalLevel {
			<-entry.continueCh // wait until done
		}
	default:
		break // avoid blocking...
	}
}

func (l *logger) Close() error {
	close(l.recCh)
	return nil
}

func (l *logger) loop() {
	for {
		select {
		case rec, ok := <-l.recCh:
			if !ok {
				// close log files
				for _, w := range l.files {
					_ = w.Close()
				}
				return
			}
			l.b.Reset()

			l.b.WriteString(time.Now().Format("2006-01-02 15:04:05"))
			l.b.WriteString(" ")
			l.b.WriteString(logLevelGlyph(rec.level))
			l.b.WriteString(" [")
			l.b.WriteString(logLevelAbbreviation(rec.level))
			l.b.WriteString("] ")

			l.b.WriteString(rec.pkg)
			if len(rec.pkg) > 0 {
				l.b.WriteString("/")
			}
			l.b.WriteString(rec.file)
			l.b.WriteString(":")
			l.b.WriteString(strconv.Itoa(rec.line))
			l.b.WriteString(" - ")
			l.b.WriteString(rec.log)
			l.b.WriteString("\n")

			line := l.b.String()

			_, _ = fmt.Fprintf(l.output, line)
			for _, w := range l.files {
				_, _ = fmt.Fprintf(w, line)
			}
			if rec.level == FatalLevel {
				exitHandler()
			}
			close(rec.continueCh)
		}
	}
}

func getCallerInfo() callerInfo {
	ci := callerInfo{}
	_, file, ln, ok := runtime.Caller(2)
	if ok {
		ci.pkg = filepath.Base(path.Dir(file))
		if ci.pkg == projectFolder {
			ci.pkg = ""
		}
		filename := filepath.Base(file)
		ci.filename = strings.TrimSuffix(filename, filepath.Ext(filename))
		ci.line = ln
	} else {
		ci.filename = "???"
		ci.pkg = "???"
	}
	return ci
}

func logLevelAbbreviation(level Level) string {
	switch level {
	case DebugLevel:
		return "DBG"
	case InfoLevel:
		return "INF"
	case WarningLevel:
		return "WRN"
	case ErrorLevel:
		return "ERR"
	case FatalLevel:
		return "FTL"
	default:
		return ""
	}
}

func logLevelGlyph(level Level) string {
	switch level {
	case DebugLevel:
		return "\U0001f50D"
	case InfoLevel:
		return "\u2139\ufe0f"
	case WarningLevel:
		return "\u26a0\ufe0f"
	case ErrorLevel:
		return "\U0001f4a5"
	case FatalLevel:
		return "\U0001f480"
	default:
		return ""
	}
}

func levelFromString(level string) (Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel, nil
	case "", "info":
		return InfoLevel, nil
	case "warning":
		return WarningLevel, nil
	case "error":
		return ErrorLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "off":
		return OffLevel, nil
	}
	return Level(-1), fmt.Errorf("log: unrecognized level: %s", level)
}
