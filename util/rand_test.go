/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package util

import (
	"encoding/hex"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandomBytes(t *testing.T) {
	rand.Seed(1234)
	r1 := hex.EncodeToString(RandomBytes(16))

	rand.Seed(3456)
	r2 := hex.EncodeToString(RandomBytes(16))

	require.Equal(t, 32, len(r1))
	require.Equal(t, 32, len(r2))
	require.Equal(t, "c28bed645434c46376369bc5cc400b4c", r1)
	require.Equal(t, "067af84b676f17b0dac36bbaa455148a", r2)
}
