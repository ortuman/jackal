/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import (
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestMockC2Stream(t *testing.T) {
	j1, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	j2, _ := jid.NewWithString("romeo@jackal.im/orchard", false)
	id := uuid.New()
	stm := NewMockC2S(id, j1)
	require.Equal(t, "ortuman", stm.Username())
	require.Equal(t, "jackal.im", stm.Domain())
	require.Equal(t, "balcony", stm.Resource())
	require.Equal(t, "ortuman@jackal.im/balcony", stm.JID().String())

	require.Equal(t, id, stm.ID())
	stm.SetJID(j2)
	require.Equal(t, "romeo@jackal.im/orchard", stm.JID().String())

	presence := xmpp.NewPresence(j1, j2, xmpp.AvailableType)
	presence.AppendElement(xmpp.NewElementName("status"))
	stm.SetPresence(presence)
	presenceElements := stm.Presence().Elements().All()
	require.Equal(t, 1, len(presenceElements))

	elem := xmpp.NewElementName("elem1234")
	stm.SendElement(elem)
	fetch := stm.FetchElement()
	require.NotNil(t, fetch)
	require.Equal(t, "elem1234", fetch.Name())

	stm.Disconnect(nil)
	require.True(t, stm.IsDisconnected())
	stm.SetSecured(true)
	require.True(t, stm.IsSecured())
	stm.SetAuthenticated(true)
	require.True(t, stm.IsAuthenticated())
}
