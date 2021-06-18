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

package c2s

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInC2SFlags_Set(t *testing.T) {
	// given
	var flgs flags

	// then
	require.False(t, flgs.isSecured())
	require.False(t, flgs.isAuthenticated())
	require.False(t, flgs.isCompressed())
	require.False(t, flgs.isBinded())
	require.False(t, flgs.isSessionStarted())

	flgs.setSecured()
	require.True(t, flgs.isSecured())
	require.False(t, flgs.isAuthenticated())
	require.False(t, flgs.isCompressed())
	require.False(t, flgs.isBinded())
	require.False(t, flgs.isSessionStarted())

	flgs.setAuthenticated()
	require.True(t, flgs.isSecured())
	require.True(t, flgs.isAuthenticated())
	require.False(t, flgs.isCompressed())
	require.False(t, flgs.isBinded())
	require.False(t, flgs.isSessionStarted())

	flgs.setCompressed()
	require.True(t, flgs.isSecured())
	require.True(t, flgs.isAuthenticated())
	require.True(t, flgs.isCompressed())
	require.False(t, flgs.isBinded())
	require.False(t, flgs.isSessionStarted())

	flgs.setBinded()
	require.True(t, flgs.isSecured())
	require.True(t, flgs.isAuthenticated())
	require.True(t, flgs.isCompressed())
	require.True(t, flgs.isBinded())
	require.False(t, flgs.isSessionStarted())

	flgs.setSessionStarted()
	require.True(t, flgs.isSecured())
	require.True(t, flgs.isAuthenticated())
	require.True(t, flgs.isCompressed())
	require.True(t, flgs.isBinded())
	require.True(t, flgs.isSessionStarted())
}
