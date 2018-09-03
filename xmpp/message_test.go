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

func TestMessageBuild(t *testing.T) {
	j, _ := jid.New("ortuman", "example.org", "balcony", false)

	elem := xmpp.NewElementName("iq")
	_, err := xmpp.NewMessageFromElement(elem, j, j) // wrong name...
	require.NotNil(t, err)

	elem.SetName("message")
	elem.SetType("invalid")
	_, err = xmpp.NewMessageFromElement(elem, j, j) // invalid type...
	require.NotNil(t, err)

	// valid message...
	elem.SetType(xmpp.ChatType)
	elem.AppendElement(xmpp.NewElementName("body"))
	message, err := xmpp.NewMessageFromElement(elem, j, j)
	require.Nil(t, err)
	require.NotNil(t, message)
	require.True(t, message.IsMessageWithBody())

	msg2 := xmpp.NewMessageType("an-id123", xmpp.GroupChatType)
	require.Equal(t, "an-id123", msg2.ID())
	require.Equal(t, xmpp.GroupChatType, msg2.Type())
}

func TestMessageType(t *testing.T) {
	message, _ := xmpp.NewMessageFromElement(xmpp.NewElementName("message"), &jid.JID{}, &jid.JID{})
	require.True(t, message.IsNormal())

	message.SetType(xmpp.NormalType)
	require.True(t, message.IsNormal())

	message.SetType(xmpp.HeadlineType)
	require.True(t, message.IsHeadline())

	message.SetType(xmpp.ChatType)
	require.True(t, message.IsChat())

	message.SetType(xmpp.GroupChatType)
	require.True(t, message.IsGroupChat())
}

func TestMessageJID(t *testing.T) {
	from, _ := jid.New("ortuman", "test.org", "balcony", false)
	to, _ := jid.New("ortuman", "example.org", "garden", false)
	message, _ := xmpp.NewMessageFromElement(xmpp.NewElementName("message"), &jid.JID{}, &jid.JID{})
	message.SetFromJID(from)
	require.Equal(t, message.FromJID().String(), message.From())
	message.SetToJID(to)
	require.Equal(t, message.ToJID().String(), message.To())
}
