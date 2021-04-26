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

package stringsutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitKeyAndValue(t *testing.T) {
	// when
	k1, v1 := SplitKeyAndValue("key=value", '=')
	k2, v2 := SplitKeyAndValue("nosep", '=')

	// then
	require.Equal(t, "key", k1)
	require.Equal(t, "value", v1)

	require.Equal(t, "", k2)
	require.Equal(t, "", v2)
}
