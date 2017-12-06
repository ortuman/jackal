/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package compress

type Compressor interface {
	Compress(b []byte) ([]byte, error)
	Uncompress(b []byte) ([]byte, error)
}
