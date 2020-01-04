/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package log

type disabledLogger struct{}

func (*disabledLogger) Level() Level {
	return OffLevel
}

func (*disabledLogger) Log(_ Level, _ string, _ string, _ int, _ string, _ ...interface{}) {}
func (*disabledLogger) Close() error                                                       { return nil }
