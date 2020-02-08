/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDialbackKey(t *testing.T) {
	secret := "s3cr3tf0rd14lb4ck"
	from := "example.org"
	to := "xmpp.example.com"
	streamID := "D60000229F"
	kg := &keyGen{secret: secret}
	require.Equal(t, "37c69b1cf07a3f67c04a5ef5902fa5114f2c76fe4a2686482ba5b89323075643", kg.generate(from, to, streamID))
}
