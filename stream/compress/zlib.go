/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package compress

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"io"
)

type ZLIBCompressor struct {
	lvl int
}

func NewZLIBCompressor(level Level) Compressor {
	z := &ZLIBCompressor{}
	switch level {
	case DefaultLevel:
		z.lvl = flate.DefaultCompression
	case BestLevel:
		z.lvl = flate.BestCompression
	case SpeedLevel:
		z.lvl = flate.BestSpeed
	}
	return z
}

func (z *ZLIBCompressor) Compress(b []byte) ([]byte, error) {
	var buff bytes.Buffer
	w, err := zlib.NewWriterLevel(&buff, z.lvl)
	if err != nil {
		return nil, err
	}
	w.Write(b)
	w.Close()
	return buff.Bytes(), nil
}

func (z *ZLIBCompressor) Uncompress(b []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	buff := bytes.NewBuffer([]byte{})
	io.Copy(buff, r)
	r.Close()
	return buff.Bytes(), nil
}
