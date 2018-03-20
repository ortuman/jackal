/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package compress

import "io"

// Compressor represents a stream compression method.
type Compressor interface {
	io.ReadWriter
}
