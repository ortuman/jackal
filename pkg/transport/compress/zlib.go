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

import (
	"compress/zlib"
	"io"
)

// ZlibCompressor represents zlib stream compressor.
type ZlibCompressor struct {
	level int
	w     io.Writer
	r     io.Reader
	zw    io.Writer
	zr    io.Reader
}

// NewZlibCompressor returns a new zlib compression method.
func NewZlibCompressor(reader io.Reader, writer io.Writer, level Level) *ZlibCompressor {
	z := &ZlibCompressor{
		w: writer,
		r: reader,
	}
	switch level {
	case DefaultCompression:
		z.level = zlib.DefaultCompression
	case BestCompression:
		z.level = zlib.BestCompression
	case SpeedCompression:
		z.level = zlib.BestSpeed
	default:
		z.level = int(level)
	}
	return z
}

func (z *ZlibCompressor) Write(p []byte) (int, error) {
	if z.zw == nil {
		zw, err := zlib.NewWriterLevel(z.w, z.level)
		if err != nil {
			return 0, err
		}
		z.zw = zw
	}
	zw := z.zw.(*zlib.Writer)
	defer func() { _ = zw.Flush() }()
	return zw.Write(p)
}

func (z *ZlibCompressor) Read(p []byte) (int, error) {
	if z.zr == nil {
		zr, err := zlib.NewReader(z.r)
		if err != nil {
			return 0, err
		}
		z.zr = zr
	}
	return z.zr.Read(p)
}
