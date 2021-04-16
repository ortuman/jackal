// Copyright 2021 The jackal Authors
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

package capsmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapabilities_HasFeature(t *testing.T) {
	// given
	cp := &Capabilities{
		Node:     "n1",
		Ver:      "v1",
		Features: []string{"f100"},
	}
	// when
	ok1 := cp.HasFeature("f10")
	ok2 := cp.HasFeature("f100")

	// then
	require.False(t, ok1)
	require.True(t, ok2)
}
