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

package instance

import (
	"errors"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func init() {
	interfaceAddresses = func() ([]net.Addr, error) {
		return []net.Addr{&net.IPNet{
			IP:   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 0, 13},
			Mask: []byte{255, 255, 255, 0},
		}}, nil
	}
}

func TestOsEnvironmentIdentifier(t *testing.T) {
	// given
	readCachedResults = false

	someUUID := "6967de42-315c-49b6-a051-0361640f961d"
	_ = os.Setenv(envInstanceID, someUUID)

	// when
	id := ID()

	// then
	require.Equal(t, someUUID, id)
}

func TestRandomIdentifier(t *testing.T) {
	// when
	readCachedResults = false

	_ = os.Setenv(envInstanceID, "")

	id := ID()

	// then
	require.True(t, len(id) > 0)
}

func TestFQDNHostname(t *testing.T) {
	// given
	_ = os.Setenv(envHostName, "xmpp1.jackal.im")
	readCachedResults = false

	// when
	hn := Hostname()

	// then
	require.Equal(t, "xmpp1.jackal.im", hn)
}

func TestIPHostname(t *testing.T) {
	// given
	_ = os.Setenv(envHostName, "")

	interfaceAddresses = func() ([]net.Addr, error) {
		return []net.Addr{&net.IPNet{
			IP:   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 192, 168, 0, 13},
			Mask: []byte{255, 255, 255, 0},
		}}, nil
	}
	readCachedResults = false

	// when
	hn := Hostname()

	// then
	require.Equal(t, "192.168.0.13", hn)
}

func TestFallbackHostname(t *testing.T) {
	// given
	_ = os.Setenv(envHostName, "")

	interfaceAddresses = func() ([]net.Addr, error) {
		return nil, errors.New("foo error")
	}
	readCachedResults = false

	// when
	hn := Hostname()

	// then
	require.Equal(t, "localhost", hn)
}
