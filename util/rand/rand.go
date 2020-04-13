/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package utilrand

import (
	"crypto/rand"
)

// RandomBytes generates a random bytes slice of length 'l'.
func RandomBytes(l int) ([]byte, error) {
	b := make([]byte, l)
	_, err := rand.Read(b)
	return b, err
}
