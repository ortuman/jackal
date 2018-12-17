/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package version_test

import (
	"testing"

	"github.com/ortuman/jackal/version"
	"github.com/stretchr/testify/require"
)

func TestNewVersion(t *testing.T) {
	v1 := version.NewVersion(1, 9, 2)
	require.Equal(t, "v1.9.2", v1.String())
}

func TestIsEqual(t *testing.T) {
	v1 := version.NewVersion(1, 9, 2)
	v2 := version.NewVersion(1, 9, 2)
	v3 := version.NewVersion(1, 8, 2)
	require.True(t, v1.IsEqual(v2))
	require.True(t, v1.IsEqual(v1))
	require.False(t, v1.IsEqual(v3))
}

func TestIsGreat(t *testing.T) {
	v1 := version.NewVersion(1, 9, 2)
	v2 := version.NewVersion(1, 9, 3)
	v3 := version.NewVersion(1, 10, 2)
	v4 := version.NewVersion(2, 9, 2)
	v5 := version.NewVersion(1, 9, 1)
	v6 := version.NewVersion(1, 9, 2)
	require.True(t, v2.IsGreater(v1))
	require.True(t, v3.IsGreater(v1))
	require.True(t, v4.IsGreater(v1))
	require.False(t, v5.IsGreater(v1))
	require.False(t, v1.IsGreater(v1))
	require.True(t, v6.IsGreaterOrEqual(v1))
}

func TestIsLess(t *testing.T) {
	v1 := version.NewVersion(1, 9, 2)
	v2 := version.NewVersion(1, 9, 1)
	v3 := version.NewVersion(1, 8, 2)
	v4 := version.NewVersion(0, 9, 2)
	v5 := version.NewVersion(1, 9, 3)
	v6 := version.NewVersion(1, 9, 2)
	require.True(t, v2.IsLess(v1))
	require.True(t, v3.IsLess(v1))
	require.True(t, v4.IsLess(v1))
	require.False(t, v5.IsLess(v1))
	require.False(t, v1.IsLess(v1))
	require.True(t, v6.IsLessOrEqual(v1))
}
