/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package zlib

import (
	"bytes"
	"fmt"

	"github.com/ortuman/jackal/stream/compress"
)

const zlibChunkSize = 4096

type ZlibCompressor struct {
	zLevel int
	wstrm  *zstream
	rstrm  *zstream
	wbuff  []byte
	rbuff  []byte
}

func NewCompressor(level compress.Level) compress.Compressor {
	z := &ZlibCompressor{}
	switch level {
	case compress.DefaultLevel:
		z.zLevel = zDefaultCompression
	case compress.BestLevel:
		z.zLevel = zBestCompression
	case compress.SpeedLevel:
		z.zLevel = zBestSpeed
	}
	return z
}

func (z *ZlibCompressor) Compress(b []byte) ([]byte, error) {
	if z.wstrm == nil {
		z.wstrm = &zstream{}
		if err := z.wstrm.deflateInit(z.zLevel); err != nil {
			return nil, err
		}
		z.wbuff = make([]byte, zlibChunkSize)
	}
	z.wstrm.setInBuf(b, len(b))
	z.wstrm.setOutBuf(z.wbuff, zlibChunkSize)

	ret := new(bytes.Buffer)
	var status, have int
	for {
		status = z.wstrm.deflate(zSyncFlush)
		if status != zOK {
			return nil, fmt.Errorf("zlib: deflate error (%d)", status)
		}
		have = zlibChunkSize - z.wstrm.availOut()

		ret.Write(z.wbuff[:have])
		if z.wstrm.availOut() != 0 {
			break
		}
	}
	return ret.Bytes(), nil
}

func (z *ZlibCompressor) Uncompress(b []byte) ([]byte, error) {
	if z.rstrm == nil {
		z.rstrm = &zstream{}
		if err := z.rstrm.inflateInit(); err != nil {
			return nil, err
		}
		z.rbuff = make([]byte, zlibChunkSize)
	}
	z.rstrm.setInBuf(b, len(b))
	z.rstrm.setOutBuf(z.rbuff, zlibChunkSize)

	ret := new(bytes.Buffer)
	var status, have int
	for {
		status = z.rstrm.inflate(zSyncFlush)
		if status != zOK {
			return nil, fmt.Errorf("zlib: inflate error (%d)", status)
		}
		have = zlibChunkSize - z.rstrm.availOut()
		fmt.Printf("have: %d\n", have)

		ret.Write(z.rbuff[:have])
		if z.rstrm.availOut() != 0 {
			break
		}
	}
	return ret.Bytes(), nil
}
