/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadCertificate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tlsCfg, err := LoadCertificate("../testdata/cert/test.server.key", "../testdata/cert/test.server.crt", "localhost")
		require.Nil(t, err)
		require.NotNil(t, tlsCfg)
	})
	t.Run("Self-Signed", func(t *testing.T) {
		tlsCfg, err := LoadCertificate("", "", "localhost")
		require.Nil(t, err)
		require.NotNil(t, tlsCfg)
	})
	t.Run("Failed", func(t *testing.T) {
		tlsCfg, err := LoadCertificate("", "", "jackal.im")
		require.Nil(t, tlsCfg)
		require.NotNil(t, err)
		require.Equal(t, "must specify a private key and a server certificate for the domain 'jackal.im'", err.Error())
	})
}
