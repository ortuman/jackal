// Copyright 2022 The jackal Authors
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

package host

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHosts_Default(t *testing.T) {
	// given
	h := &Hosts{
		hosts: make(map[string]tls.Certificate),
	}

	// when
	cer := tls.Certificate{}
	h.RegisterDefaultHost("jackal.im", cer)

	// then
	require.Equal(t, "jackal.im", h.DefaultHostName())
	require.Len(t, h.Certificates(), 1)
}

func TestHosts_Domains(t *testing.T) {
	// given
	h := &Hosts{
		hosts: make(map[string]tls.Certificate),
	}

	// when
	c1 := tls.Certificate{}
	c2 := tls.Certificate{}
	h.RegisterHost("jackal.org", c1)
	h.RegisterHost("jackal.net", c2)

	// then
	require.Len(t, h.HostNames(), 2)
	require.Len(t, h.Certificates(), 2)

	require.True(t, h.IsLocalHost("jackal.org"))
	require.True(t, h.IsLocalHost("jackal.net"))
}
