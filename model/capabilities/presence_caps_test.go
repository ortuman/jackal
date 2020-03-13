/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package capsmodel

import (
	"bytes"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestPresenceCapabilities(t *testing.T) {
	j1, _ := jid.NewWithString("ortuman@jackal.im", true)

	var p1, p2 PresenceCaps
	p1 = PresenceCaps{
		Presence: xmpp.NewPresence(j1, j1, xmpp.AvailableType),
	}

	buf := new(bytes.Buffer)
	require.Nil(t, p1.ToBytes(buf))
	require.Nil(t, p2.FromBytes(buf))
	require.Equal(t, p1, p2)

	var p3, p4 PresenceCaps
	p3 = PresenceCaps{
		Presence: xmpp.NewPresence(j1, j1, xmpp.AvailableType),
		Caps: &Capabilities{
			Node: "http://jackal.im",
			Ver:  "v1234",
		},
	}
	buf = new(bytes.Buffer)
	require.Nil(t, p3.ToBytes(buf))
	require.Nil(t, p4.FromBytes(buf))
	require.Equal(t, p3, p4)
}
