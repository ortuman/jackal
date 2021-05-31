// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"os"
	"strings"
	"sync"

	"github.com/jackal-xmpp/runqueue"
	"github.com/ortuman/jackal/pkg/cluster/instance"
	"github.com/prometheus/client_golang/prometheus"
)

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

	// OffLevel represents a disabled logger log level.
	OffLevel
)

// String returns logger's level string representation.
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarningLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	default:
		return "off"
	}
}

// Logger represents a common logger interface.
type Logger interface {
	// Debugf uses fmt.Sprintf to log a `debug` templated message.
	Debugf(msg string, args ...interface{})

	// Debugw writes a 'debug' message to configured logger with some additional context.
	Debugw(msg string, keysAndValues ...interface{})

	// Infof uses fmt.Sprintf to log an `info` templated message.
	Infof(msg string, args ...interface{})

	// Infow writes a 'info' message to configured logger with some additional context.
	Infow(msg string, keysAndValues ...interface{})

	// Warnf uses fmt.Sprintf to log a `warn` templated message.
	Warnf(msg string, args ...interface{})

	// Warnw writes a 'warning' message to configured logger with some additional context.
	Warnw(msg string, keysAndValues ...interface{})

	// Errorf uses fmt.Sprintf to log an `error` templated message.
	Errorf(msg string, args ...interface{})

	// Errorw writes an 'error' message to configured logger with some additional context.
	Errorw(msg string, keysAndValues ...interface{})

	// Fatalf uses fmt.Sprintf to log a `fatal` templated message.
	Fatalf(msg string, args ...interface{})

	// Fatalw writes a 'fatal' message to configured logger with some additional context.
	Fatalw(msg string, keysAndValues ...interface{})
}

type disabledLogger struct{}

func (l *disabledLogger) Debugw(_ string, _ ...interface{}) {}
func (l *disabledLogger) Debugf(_ string, _ ...interface{}) {}
func (l *disabledLogger) Infow(_ string, _ ...interface{})  {}
func (l *disabledLogger) Infof(_ string, _ ...interface{})  {}
func (l *disabledLogger) Warnw(_ string, _ ...interface{})  {}
func (l *disabledLogger) Warnf(_ string, _ ...interface{})  {}
func (l *disabledLogger) Errorw(_ string, _ ...interface{}) {}
func (l *disabledLogger) Errorf(_ string, _ ...interface{}) {}
func (l *disabledLogger) Fatalw(_ string, _ ...interface{}) {}
func (l *disabledLogger) Fatalf(_ string, _ ...interface{}) {}

var osExit = func() { os.Exit(-1) }

// Debugf uses fmt.Sprintf to log a `debug` templated message.
func Debugf(msg string, args ...interface{}) {
	if getLevel() > DebugLevel {
		return
	}
	logMessagef(DebugLevel, msg, args...)
}

// Debugw writes a 'debug' message to configured logger with some additional context.
func Debugw(msg string, keysAndValues ...interface{}) {
	if getLevel() > DebugLevel {
		return
	}
	logMessagew(DebugLevel, msg, keysAndValues...)
}

// Infof uses fmt.Sprintf to log an `info` templated message.
func Infof(msg string, args ...interface{}) {
	if getLevel() > InfoLevel {
		return
	}
	logMessagef(InfoLevel, msg, args...)
}

// Infow writes a 'info' message to configured logger with some additional context.
func Infow(msg string, keysAndValues ...interface{}) {
	if getLevel() > InfoLevel {
		return
	}
	logMessagew(InfoLevel, msg, keysAndValues...)
}

// Warnf uses fmt.Sprintf to log a `warn` templated message.
func Warnf(msg string, args ...interface{}) {
	if getLevel() > WarningLevel {
		return
	}
	logMessagef(WarningLevel, msg, args...)
}

// Warnw writes a 'warning' message to configured logger with some additional context.
func Warnw(msg string, keysAndValues ...interface{}) {
	if getLevel() > WarningLevel {
		return
	}
	logMessagew(WarningLevel, msg, keysAndValues...)
}

// Errorf uses fmt.Sprintf to log an `error` templated message.
func Errorf(msg string, args ...interface{}) {
	if getLevel() > ErrorLevel {
		return
	}
	logMessagef(ErrorLevel, msg, args...)
}

// Errorw writes an 'error' message to configured logger with some additional context.
func Errorw(msg string, keysAndValues ...interface{}) {
	if getLevel() > ErrorLevel {
		return
	}
	logMessagew(ErrorLevel, msg, keysAndValues...)
}

// Fatalf uses fmt.Sprintf to log a `fatal` templated message.
func Fatalf(msg string, args ...interface{}) {
	if getLevel() > FatalLevel {
		return
	}
	logMessagef(FatalLevel, msg, args...)
}

// Fatalw writes a 'fatal' message to configured logger with some additional context.
func Fatalw(msg string, keysAndValues ...interface{}) {
	if getLevel() > FatalLevel {
		return
	}
	logMessagew(FatalLevel, msg, keysAndValues...)
}

var (
	mtx   sync.RWMutex
	inst  Logger
	level Level
	rq    *runqueue.RunQueue
)

// Disabled stores a disabled logger instance.
var Disabled Logger = &disabledLogger{}

func init() {
	inst = Disabled
	level = OffLevel
}

// SetLogger sets the global logger instance.
func SetLogger(lg Logger, lgLevel string) {
	var lv = OffLevel
	switch strings.ToLower(lgLevel) {
	case "debug":
		lv = DebugLevel
	case "info":
		lv = InfoLevel
	case "warn":
		lv = WarningLevel
	case "error":
		lv = ErrorLevel
	case "fatal":
		lv = FatalLevel
	}
	mtx.Lock()
	rq = runqueue.New("logger", nil)
	inst = lg
	level = lv
	mtx.Unlock()
}

// Close stops global logger running queue.
func Close() {
	ch := make(chan bool, 1)

	mtx.RLock()
	rq.Stop(func() { close(ch) })
	<-ch
	mtx.RUnlock()
}

func logMessagef(level Level, msg string, args ...interface{}) {
	rq.Run(func() {
		inst := getInstance()
		switch level {
		case DebugLevel:
			inst.Debugf(msg, args...)
		case InfoLevel:
			inst.Infof(msg, args...)
		case WarningLevel:
			inst.Warnf(msg, args...)
		case ErrorLevel:
			inst.Errorf(msg, args...)
		case FatalLevel:
			inst.Fatalf(msg, args...)
			osExit()
		}
		loggedMessages.With(prometheus.Labels{"instance": instance.ID(), "level": level.String()})
	})
	if level == FatalLevel {
		select {} // hang forever
	}
}

func logMessagew(level Level, msg string, keysAndValues ...interface{}) {
	rq.Run(func() {
		inst := getInstance()
		switch level {
		case DebugLevel:
			inst.Debugw(msg, keysAndValues...)
		case InfoLevel:
			inst.Infow(msg, keysAndValues...)
		case WarningLevel:
			inst.Warnw(msg, keysAndValues...)
		case ErrorLevel:
			inst.Errorw(msg, keysAndValues...)
		case FatalLevel:
			inst.Fatalw(msg, keysAndValues...)
			osExit()
		}
		loggedMessages.With(prometheus.Labels{"instance": instance.ID(), "level": level.String()})
	})
}

func getInstance() Logger {
	mtx.RLock()
	l := inst
	mtx.RUnlock()
	return l
}

func getLevel() Level {
	mtx.RLock()
	lv := level
	mtx.RUnlock()
	return lv
}
