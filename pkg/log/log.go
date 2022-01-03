// Copyright 2021 The jackal Authors
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

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	debugLevel   = "debug"
	infoLevel    = "info"
	warningLevel = "warn"
	errorLevel   = "error"
	offLevel     = "off"
)

// NewDefaultLogger creates a new go-kit logger with the configured level and format.
func NewDefaultLogger(lv, format string) log.Logger {
	var logger log.Logger
	var allow level.Option

	w := log.NewSyncWriter(os.Stderr)
	if format == "json" {
		logger = log.NewJSONLogger(w)
	} else {
		logger = log.NewLogfmtLogger(w)
	}
	switch lv {
	case debugLevel:
		allow = level.AllowDebug()
	case infoLevel:
		allow = level.AllowInfo()
	case warningLevel:
		allow = level.AllowWarn()
	case errorLevel:
		allow = level.AllowError()
	case offLevel:
		allow = level.AllowNone()
	default:
		allow = level.AllowAll()
	}
	return log.With(level.NewFilter(logger, allow), "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
}
