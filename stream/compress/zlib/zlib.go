/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package zlib

import (
	"fmt"
	"io"

	"bytes"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/stream/compress"
)

const zlibChunkSize = 4096

type ZlibCompressor struct {
	zLevel int
	w      io.Writer
	wstrm  *zstream
	wbuff  []byte
	r      io.Reader
	rstrm  *zstream
	rbuff  []byte
}

func NewCompressor(reader io.Reader, writer io.Writer, level config.CompressionLevel) compress.Compressor {
	z := &ZlibCompressor{}
	switch level {
	case config.DefaultCompression:
		z.zLevel = zDefaultCompression
	case config.BestCompression:
		z.zLevel = zBestCompression
	case config.SpeedCompression:
		z.zLevel = zBestSpeed
	}
	z.r = reader
	z.w = writer
	return z
}

func (z *ZlibCompressor) Write(p []byte) (int, error) {
	if z.wstrm == nil {
		z.wstrm = &zstream{}
		if err := z.wstrm.deflateInit(z.zLevel); err != nil {
			return 0, err
		}
		z.wbuff = make([]byte, zlibChunkSize)
	}
	z.wstrm.setInBuf(p, len(p))
	z.wstrm.setOutBuf(z.wbuff, zlibChunkSize)

	var status, have, n int
	for {
		status = z.wstrm.deflate(zSyncFlush)
		if status != zOK {
			return n, fmt.Errorf("zlib: deflate error (%d)", status)
		}
		have = zlibChunkSize - z.wstrm.availOut()

		wn, err := z.w.Write(z.wbuff[:have])
		if err != nil {
			return n, err
		}
		n += wn

		if z.wstrm.availOut() != 0 {
			break
		}
	}
	return n, nil
}

func (z *ZlibCompressor) Read(p []byte) (int, error) {
	if z.rstrm == nil {
		z.rstrm = &zstream{}
		if err := z.rstrm.inflateInit(); err != nil {
			return 0, err
		}
		z.rbuff = make([]byte, zlibChunkSize)
	}
	rbuf := make([]byte, len(p))
	rn, err := z.r.Read(rbuf)
	if err != nil {
		return rn, err
	}
	z.rstrm.setInBuf(rbuf, len(rbuf))
	z.rstrm.setOutBuf(z.rbuff, zlibChunkSize)

	outBuf := bytes.NewBuffer(make([]byte, len(p)))
	var status, have, n int
	for {
		status = z.rstrm.inflate(zSyncFlush)
		if status != zOK {
			return n, fmt.Errorf("zlib: inflate error (%d)", status)
		}
		have = zlibChunkSize - z.rstrm.availOut()

		outBuf.Write(z.rbuff[:have])
		if z.rstrm.availOut() != 0 {
			break
		}
	}
	return outBuf.Read(p)
}
