/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapabilities(t *testing.T) {
	var c1, c2 Capabilities
	c1 = Capabilities{[]string{"ns"}}
	buf := new(bytes.Buffer)
	require.Nil(t, c1.ToBytes(buf))
	require.Nil(t, c2.FromBytes(buf))
	require.Equal(t, c1, c2)
}
