/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package rostermodel

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelRosterVersion(t *testing.T) {
	var rv1, rv2 Version
	rv1 = Version{Ver: 2, DeletionVer: 1}
	buf := new(bytes.Buffer)
	require.Nil(t, rv1.ToBytes(buf))
	require.Nil(t, rv2.FromBytes(buf))
	require.Equal(t, rv1, rv2)
}
