/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeStrings(t *testing.T) {
	require.Equal(t, "socket", Socket.String())
	require.Equal(t, "", Type(99).String())
}
