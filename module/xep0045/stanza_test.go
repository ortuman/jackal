/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0045

import (
	"testing"

	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestNewItem(t *testing.T) {
	item := newItemElement("owner", "moderator")
	require.Equal(t, item.Name(), "item")
	require.Equal(t, item.Attributes().Get("affiliation"), "owner")
	require.Equal(t, item.Attributes().Get("role"), "moderator")

}

func TestNewStatus(t *testing.T) {
	status := newStatusElement("200")
	require.Equal(t, status.Name(), "status")
	require.Equal(t, status.Attributes().Get("code"), "200")
}

func TestGetAck(t *testing.T) {
	from, _ := jid.New("ortuman", "test.org", "balcony", false)
	to, _ := jid.New("ortuman", "example.org", "garden", false)
	message := getAckStanza(from, to)
	require.Equal(t, message.Name(), "presence")
	require.Equal(t, message.From(), from.String())
	require.Equal(t, message.To(), to.String())

	xel := message.Elements().Child("x")
	require.Equal(t, xel.Namespace(), mucNamespaceUser)
	require.Equal(t, xel.Elements().Child("item").String(),
		newItemElement("owner", "moderator").String())
}
