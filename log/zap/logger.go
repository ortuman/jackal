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

package zap

import (
	"github.com/ortuman/jackal/cluster/instance"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger represents a zap logger implementation.
type Logger struct {
	lg       *zap.Logger
	sgLogger *zap.SugaredLogger
}

// NewLogger creates an initialized zap logger instance.
func NewLogger(outputPath string) *Logger {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	cfg.DisableCaller = true
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	cfg.InitialFields = map[string]interface{}{
		"instance_id": instance.ID(),
	}

	outputPaths := []string{"/dev/stdout"}
	if len(outputPath) > 0 {
		outputPaths = append(outputPaths, outputPath)
	}
	cfg.OutputPaths = outputPaths

	logger, _ := cfg.Build()
	sugaredLogger := logger.Sugar()
	return &Logger{
		lg:       logger,
		sgLogger: sugaredLogger,
	}
}

// Debugf uses fmt.Sprintf to log a `debug` templated message.
func (l *Logger) Debugf(msg string, args ...interface{}) {
	l.sgLogger.Debugf(msg, args...)
	_ = l.lg.Sync()
}

// Debugw writes a 'debug' message to configured logger with some additional context.
func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.sgLogger.Debugw(msg, keysAndValues...)
	_ = l.lg.Sync()
}

// Infof uses fmt.Sprintf to log an `info` templated message.
func (l *Logger) Infof(msg string, args ...interface{}) {
	l.sgLogger.Infof(msg, args...)
	_ = l.lg.Sync()
}

// Infow writes a 'info' message to configured logger with some additional context.
func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.sgLogger.Infow(msg, keysAndValues...)
	_ = l.lg.Sync()
}

// Warnf uses fmt.Sprintf to log a `warn` templated message.
func (l *Logger) Warnf(msg string, args ...interface{}) {
	l.sgLogger.Warnf(msg, args...)
	_ = l.lg.Sync()
}

// Warnw writes a 'warning' message to configured logger with some additional context.
func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.sgLogger.Warnw(msg, keysAndValues...)
	_ = l.lg.Sync()
}

// Errorf uses fmt.Sprintf to log an `error` templated message.
func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.sgLogger.Errorf(msg, args...)
	_ = l.lg.Sync()
}

// Errorw writes an 'error' message to configured logger with some additional context.
func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.sgLogger.Errorw(msg, keysAndValues...)
	_ = l.lg.Sync()
}

// Fatalf uses fmt.Sprintf to log a `fatal` templated message.
func (l *Logger) Fatalf(msg string, args ...interface{}) {
	l.sgLogger.Fatalf(msg, args...)
	_ = l.lg.Sync()
}

// Fatalw writes a 'fatal' message to configured logger with some additional context.
func (l *Logger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.sgLogger.Fatalw(msg, keysAndValues...)
	_ = l.lg.Sync()
}
