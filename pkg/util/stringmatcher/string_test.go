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

package stringmatcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatcher_String(t *testing.T) {
	// given
	m := NewStringMatcher([]string{"s0", "s10", "s100"})

	// when
	r0 := m.Matches("s0")
	r1 := m.Matches("s1")
	r2 := m.Matches("s10")
	r3 := m.Matches("s101")
	r4 := m.Matches("s100")
	r5 := m.Matches("s1000")

	// then
	require.True(t, r0)
	require.False(t, r1)
	require.True(t, r2)
	require.False(t, r3)
	require.True(t, r4)
	require.False(t, r5)
}
