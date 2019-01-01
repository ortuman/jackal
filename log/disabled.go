/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package log

type disabledLogger struct{}

func (*disabledLogger) Level() Level {
	return OffLevel
}

func (*disabledLogger) Log(level Level, pkg string, file string, line int, format string, args ...interface{}) {
}

func (*disabledLogger) Close() error { return nil }
