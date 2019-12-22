/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp_test

import (
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestPresenceBuild(t *testing.T) {
	j, _ := jid.New("ortuman", "example.org", "balcony", false)

	elem := xmpp.NewElementName("message")
	_, err := xmpp.NewPresenceFromElement(elem, j, j) // wrong name...
	require.NotNil(t, err)

	// invalid type
	elem.SetName("presence")
	elem.SetType("invalid")
	_, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	// test show
	elem.SetType(xmpp.AvailableType)
	presence, err := xmpp.NewPresenceFromElement(elem, j, j)
	require.Nil(t, err)
	require.Equal(t, presence.ShowState(), xmpp.AvailableShowState)

	show := xmpp.NewElementName("show")
	show.SetText("invalid")
	elem.AppendElement(show)
	_, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	ss := []string{"away", "chat", "dnd", "xa"}
	expected := []xmpp.ShowState{xmpp.AwayShowState, xmpp.ChatShowState, xmpp.DoNotDisturbShowState, xmpp.ExtendedAwaysShowState}
	for i, showState := range ss {
		elem.ClearElements()

		show := xmpp.NewElementName("show")
		show.SetText(showState)
		elem.AppendElement(show)
		presence, err := xmpp.NewPresenceFromElement(elem, j, j)
		require.Nil(t, err)
		require.Equal(t, expected[i], presence.ShowState())
	}

	// show with attribute
	elem.ClearElements()
	show = xmpp.NewElementNamespace("show", "ns")
	elem.AppendElement(show)
	_, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	// show > 1
	elem.ClearElements()
	show1 := xmpp.NewElementName("show")
	show2 := xmpp.NewElementName("show")
	elem.AppendElement(show1)
	elem.AppendElement(show2)
	_, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	// test priority
	elem.ClearElements()
	priority := xmpp.NewElementName("priority")
	priority2 := xmpp.NewElementName("priority")
	elem.AppendElement(priority)
	elem.AppendElement(priority2)
	_, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	elem.ClearElements()
	priority.SetText("abcd")
	elem.AppendElement(priority)
	_, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	elem.ClearElements()
	priority.SetText("129")
	elem.AppendElement(priority)
	_, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	elem.ClearElements()
	priority.SetText("120")
	elem.AppendElement(priority)
	presence, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.Nil(t, err)
	require.Equal(t, int8(120), presence.Priority())

	// test status
	elem.ClearElements()
	status := xmpp.NewElementNamespace("status", "ns")
	elem.AppendElement(status)
	_, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	elem.ClearElements()
	status = xmpp.NewElementName("status")
	status.SetLanguage("en")
	status.SetText("Readable text")
	elem.AppendElement(status)
	presence, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.Nil(t, err)
	require.Equal(t, "Readable text", presence.Status())

	elem.ClearElements()
	status.RemoveAttribute("xml:lang")
	elem.AppendElement(status)
	presence, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.Nil(t, err)
	require.Equal(t, "Readable text", presence.Status())

	elem.ClearElements()
	c := xmpp.NewElementNamespace("c", "http://jabber.org/protocol/caps")
	c.SetAttribute("hash", "sha-1")
	c.SetAttribute("node", "http://code.google.com/p/exodus")
	c.SetAttribute("ver", "QgayPKawpkPSDYmwT/WM94uAlu0=")
	elem.AppendElement(c)
	presence, err = xmpp.NewPresenceFromElement(elem, j, j)
	require.Nil(t, err)

	caps := presence.Capabilities()
	require.NotNil(t, caps)
	require.Equal(t, "sha-1", caps.Hash)
	require.Equal(t, "http://code.google.com/p/exodus", caps.Node)
	require.Equal(t, "QgayPKawpkPSDYmwT/WM94uAlu0=", caps.Ver)
}

func TestPresenceType(t *testing.T) {
	presence := xmpp.NewPresence(&jid.JID{}, &jid.JID{}, "")
	require.True(t, presence.IsAvailable())

	presence.SetType(xmpp.AvailableType)
	require.True(t, presence.IsAvailable())

	presence.SetType(xmpp.UnavailableType)
	require.True(t, presence.IsUnavailable())

	presence.SetType(xmpp.SubscribeType)
	require.True(t, presence.IsSubscribe())

	presence.SetType(xmpp.SubscribedType)
	require.True(t, presence.IsSubscribed())

	presence.SetType(xmpp.UnsubscribeType)
	require.True(t, presence.IsUnsubscribe())

	presence.SetType(xmpp.UnsubscribedType)
	require.True(t, presence.IsUnsubscribed())
}

func TestPresenceJID(t *testing.T) {
	from, _ := jid.New("ortuman", "test.org", "balcony", false)
	to, _ := jid.New("ortuman", "example.org", "garden", false)
	presence, _ := xmpp.NewPresenceFromElement(xmpp.NewElementName("presence"), &jid.JID{}, &jid.JID{})
	presence.SetFromJID(from)
	require.Equal(t, presence.FromJID().String(), presence.From())
	presence.SetToJID(to)
	require.Equal(t, presence.ToJID().String(), presence.To())
}
