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

package instance

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOsEnvironmentIdentifier(t *testing.T) {
	// given
	someUUID := "6967de42-315c-49b6-a051-0361640f961d"
	_ = os.Setenv(envInstanceID, someUUID)

	// when
	id := ID()

	// then
	require.Equal(t, someUUID, id)
}

func TestRandomIdentifier(t *testing.T) {
	// when
	_ = os.Setenv(envInstanceID, "")

	id := ID()

	// then
	require.True(t, len(id) > 0)
}
