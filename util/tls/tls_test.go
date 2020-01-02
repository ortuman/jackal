/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package utiltls

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
		defer os.RemoveAll(".cert/")

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
