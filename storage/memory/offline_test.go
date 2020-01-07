/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertOfflineMessage(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	message := xmpp.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xmpp.NewElementName("body"))
	m, _ := xmpp.NewMessageFromElement(message, j, j)

	s := NewOffline()
	EnableMockedError()
	require.Equal(t, ErrMocked, s.InsertOfflineMessage(context.Background(), m, "ortuman"))
	DisableMockedError()

	require.Nil(t, s.InsertOfflineMessage(context.Background(), m, "ortuman"))
}

func TestMemoryStorage_CountOfflineMessages(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	message := xmpp.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xmpp.NewElementName("body"))
	m, _ := xmpp.NewMessageFromElement(message, j, j)

	s := NewOffline()
	_ = s.InsertOfflineMessage(context.Background(), m, "ortuman")

	EnableMockedError()
	_, err := s.CountOfflineMessages(context.Background(), "ortuman")
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	cnt, _ := s.CountOfflineMessages(context.Background(), "ortuman")
	require.Equal(t, 1, cnt)
}

func TestMemoryStorage_FetchOfflineMessages(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	message := xmpp.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xmpp.NewElementName("body"))
	m, _ := xmpp.NewMessageFromElement(message, j, j)

	s := NewOffline()
	_ = s.InsertOfflineMessage(context.Background(), m, "ortuman")

	EnableMockedError()
	_, err := s.FetchOfflineMessages(context.Background(), "ortuman")
	require.Equal(t, ErrMocked, err)
	DisableMockedError()
	elems, _ := s.FetchOfflineMessages(context.Background(), "ortuman")
	require.Equal(t, 1, len(elems))
}

func TestMemoryStorage_DeleteOfflineMessages(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	message := xmpp.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xmpp.NewElementName("body"))
	m, _ := xmpp.NewMessageFromElement(message, j, j)

	s := NewOffline()
	_ = s.InsertOfflineMessage(context.Background(), m, "ortuman")

	EnableMockedError()
	require.Equal(t, ErrMocked, s.DeleteOfflineMessages(context.Background(), "ortuman"))
	DisableMockedError()
	require.Nil(t, s.DeleteOfflineMessages(context.Background(), "ortuman"))

	elems, _ := s.FetchOfflineMessages(context.Background(), "ortuman")
	require.Equal(t, 0, len(elems))
}
