/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package compress

import "io"

type Compressor interface {
	io.ReadWriter
}
