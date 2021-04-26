// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version_test

import (
	"testing"

	"github.com/ortuman/jackal/pkg/version"
	"github.com/stretchr/testify/require"
)

func TestNewVersion(t *testing.T) {
	v1 := version.NewVersion(1, 9, 2)

	require.Equal(t, uint(1), v1.Major())
	require.Equal(t, uint(9), v1.Minor())
	require.Equal(t, uint(2), v1.Patch())
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
