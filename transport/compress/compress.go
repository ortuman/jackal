/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package compress

import "io"

// Level represents a stream compression level.
type Level int

const (
	// NoCompression represents no stream compression.
	NoCompression Level = iota

	// DefaultCompression represents 'default' stream compression level.
	DefaultCompression

	// BestCompression represents 'best for size' stream compression level.
	BestCompression

	// SpeedCompression represents 'best for speed' stream compression level.
	SpeedCompression
)

// String returns CompressionLevel string representation.
func (cl Level) String() string {
	switch cl {
	case DefaultCompression:
		return "default"
	case BestCompression:
		return "best"
	case SpeedCompression:
		return "speed"
	}
	return ""
}

// Compressor represents a stream compression method.
type Compressor interface {
	io.ReadWriter
}
