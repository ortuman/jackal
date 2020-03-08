/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package capsmodel

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapabilities(t *testing.T) {
	var c1, c2 Capabilities
	c1 = Capabilities{Node: "n", Ver: "v", Features: []string{"ns1", "ns2"}}

	require.True(t, c1.HasFeature("ns2"))
	require.False(t, c1.HasFeature("ns3"))

	buf := new(bytes.Buffer)
	require.Nil(t, c1.ToBytes(buf))
	require.Nil(t, c2.FromBytes(buf))
	require.Equal(t, c1, c2)
}
