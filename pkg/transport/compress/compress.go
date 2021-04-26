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
	case NoCompression:
		return "no_compression"
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
