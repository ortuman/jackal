/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml_test

import (
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

func TestPresenceBuild(t *testing.T) {
	j, _ := xml.NewJID("ortuman", "example.org", "balcony", false)

	elem := xml.NewElementName("message")
	_, err := xml.NewPresenceFromElement(elem, j, j) // wrong name...
	require.NotNil(t, err)

	// invalid type
	elem.SetName("presence")
	elem.SetType("invalid")
	_, err = xml.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	// test show
	elem.SetType(xml.AvailableType)
	presence, err := xml.NewPresenceFromElement(elem, j, j)
	require.Nil(t, err)
	require.Equal(t, presence.ShowState(), xml.AvailableShowState)

	show := xml.NewElementName("show")
	show.SetText("invalid")
	elem.AppendElement(show)
	_, err = xml.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	ss := []string{"away", "chat", "dnd", "xa"}
	expected := []xml.ShowState{xml.AwayShowState, xml.ChatShowState, xml.DoNotDisturbShowState, xml.ExtendedAwaysShowState}
	for i, showState := range ss {
		elem.ClearElements()

		show := xml.NewElementName("show")
		show.SetText(showState)
		elem.AppendElement(show)
		presence, err := xml.NewPresenceFromElement(elem, j, j)
		require.Nil(t, err)
		require.Equal(t, expected[i], presence.ShowState())
	}

	// show with attribute
	elem.ClearElements()
	show = xml.NewElementNamespace("show", "ns")
	elem.AppendElement(show)
	_, err = xml.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	// show > 1
	elem.ClearElements()
	show1 := xml.NewElementName("show")
	show2 := xml.NewElementName("show")
	elem.AppendElement(show1)
	elem.AppendElement(show2)
	_, err = xml.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	// test priority
	elem.ClearElements()
	priority := xml.NewElementName("priority")
	priority2 := xml.NewElementName("priority")
	elem.AppendElement(priority)
	elem.AppendElement(priority2)
	_, err = xml.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	elem.ClearElements()
	priority.SetText("abcd")
	elem.AppendElement(priority)
	_, err = xml.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	elem.ClearElements()
	priority.SetText("129")
	elem.AppendElement(priority)
	_, err = xml.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	elem.ClearElements()
	priority.SetText("120")
	elem.AppendElement(priority)
	presence, err = xml.NewPresenceFromElement(elem, j, j)
	require.Nil(t, err)
	require.Equal(t, int8(120), presence.Priority())

	// test status
	elem.ClearElements()
	status := xml.NewElementNamespace("status", "ns")
	elem.AppendElement(status)
	_, err = xml.NewPresenceFromElement(elem, j, j)
	require.NotNil(t, err)

	elem.ClearElements()
	status = xml.NewElementName("status")
	status.SetLanguage("en")
	status.SetText("Readable text")
	elem.AppendElement(status)
	presence, err = xml.NewPresenceFromElement(elem, j, j)
	require.Nil(t, err)
	require.Equal(t, "Readable text", presence.Status())

	elem.ClearElements()
	status.RemoveAttribute("xml:lang")
	elem.AppendElement(status)
	presence, err = xml.NewPresenceFromElement(elem, j, j)
	require.Nil(t, err)
	require.Equal(t, "Readable text", presence.Status())
}

func TestPresenceType(t *testing.T) {
	presence := xml.NewPresence(&xml.JID{}, &xml.JID{}, "")
	require.True(t, presence.IsAvailable())

	presence.SetType(xml.AvailableType)
	require.True(t, presence.IsAvailable())

	presence.SetType(xml.UnavailableType)
	require.True(t, presence.IsUnavailable())

	presence.SetType(xml.SubscribeType)
	require.True(t, presence.IsSubscribe())

	presence.SetType(xml.SubscribedType)
	require.True(t, presence.IsSubscribed())

	presence.SetType(xml.UnsubscribeType)
	require.True(t, presence.IsUnsubscribe())

	presence.SetType(xml.UnsubscribedType)
	require.True(t, presence.IsUnsubscribed())
}

func TestPresenceJID(t *testing.T) {
	from, _ := xml.NewJID("ortuman", "test.org", "balcony", false)
	to, _ := xml.NewJID("ortuman", "example.org", "garden", false)
	presence, _ := xml.NewPresenceFromElement(xml.NewElementName("presence"), &xml.JID{}, &xml.JID{})
	presence.SetFromJID(from)
	require.Equal(t, presence.FromJID().String(), presence.From())
	presence.SetToJID(to)
	require.Equal(t, presence.ToJID().String(), presence.To())
}
