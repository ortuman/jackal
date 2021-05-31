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

package crashreporter

import (
	syslog "log"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/sentry-go"
)

const (
	envSentryDSN = "JACKAL_SENTRY_DSN"

	depthForRecoverAndReportPanic = 3
)

var crashReporterEnabled bool

func init() {
	sentryDSN := os.Getenv(envSentryDSN)
	if len(sentryDSN) == 0 {
		return
	}
	if err := sentry.Init(sentry.ClientOptions{Dsn: sentryDSN}); err != nil {
		syslog.Printf("sentry.Init: %s", err)
		return
	}
	crashReporterEnabled = true
}

func RecoverAndReportPanic() {
	if r := recover(); r != nil {
		panicErr := panicAsError(depthForRecoverAndReportPanic+1, r)
		if crashReporterEnabled {
			sendCrashReport(panicErr)
		}
		syslog.Fatalf("A panic has occurred!\n%+v", panicErr)
	}
}

func panicAsError(depth int, r interface{}) error {
	if err, ok := r.(error); ok {
		return errors.WithStackDepth(err, depth+1)
	}
	return errors.NewWithDepthf(depth+1, "panic: %v", r)
}

func sendCrashReport(err error) {
	event, extraDetails := errors.BuildSentryReport(err)

	for extraKey, extraValue := range extraDetails {
		event.Extra[extraKey] = extraValue
	}
	event.ServerName = "<redacted>"
	event.Tags["report_type"] = "panic"

	_ = sentry.CaptureEvent(event)
	_ = sentry.Flush(10 * time.Second)
}
