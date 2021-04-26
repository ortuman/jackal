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

package tlsutil

import (
	"crypto/tls"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadCertificate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tlsCfg, err := LoadCertificate("../../testdata/cert/test.server.key", "../../testdata/cert/test.server.crt", "localhost")
		require.Nil(t, err)
		require.NotNil(t, tlsCfg)
	})
	t.Run("SelfSigned", func(t *testing.T) {
		defer func() { _ = os.RemoveAll(".cert/") }()

		tlsCfg, err := LoadCertificate("", "", "localhost")
		require.Nil(t, err)
		require.NotNil(t, tlsCfg)
	})
	t.Run("Failed", func(t *testing.T) {
		cer, err := LoadCertificate("", "", "jackal.im")
		require.Equal(t, tls.Certificate{}, cer)
		require.NotNil(t, err)
		require.Equal(t, "must specify a private key and a server certificate for the domain 'jackal.im'", err.Error())
	})
}
