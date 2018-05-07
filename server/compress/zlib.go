/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

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
	defer zw.Flush()
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
