/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package host

import (
	"os"
	"testing"

	"github.com/ortuman/jackal/util"
	"github.com/stretchr/testify/require"
)

func TestHostInitialize(t *testing.T) {
	Initialize(nil)
	require.True(t, IsLocalHost("localhost"))
	require.False(t, IsLocalHost("jackal.im"))
	os.RemoveAll("./.cert")
	Shutdown()

	Initialize([]Config{{Name: "jackal.im"}})
	require.False(t, IsLocalHost("localhost"))
	require.True(t, IsLocalHost("jackal.im"))
	Shutdown()

	privKeyFile := "../testdata/cert/test.server.key"
	certFile := "../testdata/cert/test.server.crt"
	cer, err := util.LoadCertificate(privKeyFile, certFile, "localhost")
	require.Nil(t, err)

	Initialize([]Config{{Name: "localhost", Certificate: cer}})
	require.Equal(t, 1, len(Certificates()))
}
