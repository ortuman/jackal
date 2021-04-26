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

package pepper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func init() {
	minKeyLength = 1
}

func TestKeys_Get(t *testing.T) {
	// given
	ks, _ := NewKeys(map[string]string{
		"v1": "k1",
		"v2": "k2",
		"v3": "k3",
	}, "v2")

	// then
	require.Equal(t, "k2", ks.GetActiveKey())
	require.Equal(t, "k3", ks.GetKey("v3"))
	require.Equal(t, "k1", ks.GetKey("v1"))

	require.Equal(t, "v2", ks.GetActiveID())
}

func TestKeys_Error(t *testing.T) {
	// given
	_, err1 := NewKeys(map[string]string{
		"v1": "k1",
		"v2": "k2",
		"v3": "k3",
	}, "v4")

	_, err2 := NewKeys(map[string]string{}, "v4")

	// then
	require.NotNil(t, err1)
	require.NotNil(t, err2)
}
