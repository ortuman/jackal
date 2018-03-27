/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestMockC2Stream(t *testing.T) {
	j1, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	j2, _ := xml.NewJIDString("romeo@jackal.im/orchard", false)
	strm := NewMockStream(uuid.New(), j1)
	require.Equal(t, "ortuman", strm.Username())
	require.Equal(t, "jackal.im", strm.Domain())
	require.Equal(t, "balcony", strm.Resource())
	require.Equal(t, "ortuman@jackal.im/balcony", strm.JID().String())

	id := uuid.New()
	strm.SetID(id)
	require.Equal(t, id, strm.ID())
	strm.SetUsername("juliet")
	require.Equal(t, "juliet", strm.Username())
	strm.SetDomain("jackal.im")
	require.Equal(t, "jackal.im", strm.Domain())
	strm.SetResource("garden")
	require.Equal(t, "garden", strm.Resource())
	strm.SetJID(j2)
	require.Equal(t, "romeo@jackal.im/orchard", strm.JID().String())
	strm.SetPriority(-10)
	require.Equal(t, int8(-10), strm.Priority())

	strm.Disconnect(nil)
	require.True(t, strm.IsDisconnected())
	strm.SetSecured(true)
	require.True(t, strm.IsSecured())
	strm.SetCompressed(true)
	require.True(t, strm.IsCompressed())
	strm.SetAuthenticated(true)
	require.True(t, strm.IsAuthenticated())
	strm.SetRosterRequested(true)
	require.True(t, strm.IsRosterRequested())

	status := xml.NewElementName("status")
	strm.SetPresenceElements([]xml.ElementNode{status})
	presenceElements := strm.PresenceElements()
	require.Equal(t, 1, len(presenceElements))

	elem := xml.NewElementName("elem1234")
	strm.SendElement(elem)
	fetch := strm.FetchElement()
	require.NotNil(t, fetch)
	require.Equal(t, "elem1234", fetch.Name())
}
