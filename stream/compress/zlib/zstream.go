// Copyright 2015, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zlib

/*
#cgo LDFLAGS: -lz
#cgo CFLAGS: -Werror=implicit

#include "zlib.h"

int zlibstream_inflate_init(char *strm) {
  ((z_stream*)strm)->zalloc = Z_NULL;
  ((z_stream*)strm)->zfree = Z_NULL;
  ((z_stream*)strm)->opaque = Z_NULL;
  ((z_stream*)strm)->avail_in = 0;
  ((z_stream*)strm)->next_in = Z_NULL;
  return inflateInit((z_stream*)strm);
}

int zlibstream_deflate_init(char *strm, int level) {
  ((z_stream*)strm)->zalloc = Z_NULL;
  ((z_stream*)strm)->zfree = Z_NULL;
  ((z_stream*)strm)->opaque = Z_NULL;
  ((z_stream*)strm)->avail_in = 0;
  ((z_stream*)strm)->next_in = Z_NULL;
  return deflateInit((z_stream*)strm, level);
}

unsigned int zlibstream_avail_in(char *strm) {
  return ((z_stream*)strm)->avail_in;
}

unsigned int zlibstream_avail_out(char *strm) {
  return ((z_stream*)strm)->avail_out;
}

char* zlibstream_msg(char *strm) {
  return ((z_stream*)strm)->msg;
}

void zlibstream_set_in_buf(char *strm, void *buf, unsigned int len) {
  ((z_stream*)strm)->next_in = (Bytef*)buf;
  ((z_stream*)strm)->avail_in = len;
}

void zlibstream_set_out_buf(char *strm, void *buf, unsigned int len) {
  ((z_stream*)strm)->next_out = (Bytef*)buf;
  ((z_stream*)strm)->avail_out = len;
}

int zlibstream_inflate(char *strm, int flag) {
  return inflate((z_stream*)strm, flag);
}

int zlibstream_deflate(char *strm, int flag) {
  return deflate((z_stream*)strm, flag);
}

void zlibstream_inflate_end(char *strm) {
  inflateEnd((z_stream*)strm);
}

void zlibstream_deflate_end(char *strm) {
  deflateEnd((z_stream*)strm);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

const (
	// Allowed flush values
	zNoFlush      = 0
	zPartialFlush = 1
	zSyncFlush    = 2
	zFullFlush    = 3
	zFinish       = 4
	zBlock        = 5
	zTrees        = 6

	// Return codes
	zOK           = 0
	zStreamEnd    = 1
	zNeedDict     = 2
	zErrno        = -1
	zStreamError  = -2
	zDataError    = -3
	zMemError     = -4
	zBufError     = -5
	zVersionError = -6

	// compression levels
	zNoCompression      = 0
	zBestSpeed          = 1
	zBestCompression    = 9
	zDefaultCompression = -1
)

// z_stream is a buffer that's big enough to fit a C.z_stream.
// This lets us allocate a C.z_stream within Go, while keeping the contents
// opaque to the Go GC. Otherwise, the GC would look inside and complain that
// the pointers are invalid, since they point to objects allocated by C code.
type zstream [unsafe.Sizeof(C.z_stream{})]C.char

func (strm *zstream) inflateInit() error {
	result := C.zlibstream_inflate_init(&strm[0])
	if result != C.Z_OK {
		return fmt.Errorf("zlib: failed to initialize inflate (%v): %v", result, strm.msg())
	}
	return nil
}

func (strm *zstream) deflateInit(level int) error {
	result := C.zlibstream_deflate_init(&strm[0], C.int(level))
	if result != C.Z_OK {
		return fmt.Errorf("zlib: failed to initialize deflate (%v): %v", result, strm.msg())
	}
	return nil
}

func (strm *zstream) inflateEnd() {
	C.zlibstream_inflate_end(&strm[0])
}

func (strm *zstream) deflateEnd() {
	C.zlibstream_deflate_end(&strm[0])
}

func (strm *zstream) availIn() int {
	return int(C.zlibstream_avail_in(&strm[0]))
}

func (strm *zstream) availOut() int {
	return int(C.zlibstream_avail_out(&strm[0]))
}

func (strm *zstream) msg() string {
	return C.GoString(C.zlibstream_msg(&strm[0]))
}

func (strm *zstream) setInBuf(buf []byte, size int) {
	if buf == nil {
		C.zlibstream_set_in_buf(&strm[0], nil, C.uint(size))
	} else {
		C.zlibstream_set_in_buf(&strm[0], unsafe.Pointer(&buf[0]), C.uint(size))
	}
}

func (strm *zstream) setOutBuf(buf []byte, size int) {
	if buf == nil {
		C.zlibstream_set_out_buf(&strm[0], nil, C.uint(size))
	} else {
		C.zlibstream_set_out_buf(&strm[0], unsafe.Pointer(&buf[0]), C.uint(size))
	}
}

func (strm *zstream) inflate(flag int) int {
	return int(C.zlibstream_inflate(&strm[0], C.int(flag)))
}

func (strm *zstream) deflate(flag int) int {
	return int(C.zlibstream_deflate(&strm[0], C.int(flag)))
}
