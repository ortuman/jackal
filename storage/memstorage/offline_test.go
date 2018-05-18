/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestMockStorageInsertOfflineMessage(t *testing.T) {
	j, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	message := xml.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xml.NewElementName("body"))
	m, _ := xml.NewMessageFromElement(message, j, j)

	s := New()
	s.ActivateMockedError()
	require.Equal(t, ErrMockedError, s.InsertOfflineMessage(m, "ortuman"))
	s.DeactivateMockedError()
	require.Nil(t, s.InsertOfflineMessage(m, "ortuman"))
}

func TestMockStorageCountOfflineMessages(t *testing.T) {
	j, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	message := xml.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xml.NewElementName("body"))
	m, _ := xml.NewMessageFromElement(message, j, j)

	s := New()
	s.InsertOfflineMessage(m, "ortuman")

	s.ActivateMockedError()
	_, err := s.CountOfflineMessages("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DeactivateMockedError()
	cnt, _ := s.CountOfflineMessages("ortuman")
	require.Equal(t, 1, cnt)
}

func TestMockStorageFetchOfflineMessages(t *testing.T) {
	j, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	message := xml.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xml.NewElementName("body"))
	m, _ := xml.NewMessageFromElement(message, j, j)

	s := New()
	s.InsertOfflineMessage(m, "ortuman")

	s.ActivateMockedError()
	_, err := s.FetchOfflineMessages("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DeactivateMockedError()
	elems, _ := s.FetchOfflineMessages("ortuman")
	require.Equal(t, 1, len(elems))
}

func TestMockStorageDeleteOfflineMessages(t *testing.T) {
	j, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	message := xml.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xml.NewElementName("body"))
	m, _ := xml.NewMessageFromElement(message, j, j)

	s := New()
	s.InsertOfflineMessage(m, "ortuman")

	s.ActivateMockedError()
	require.Equal(t, ErrMockedError, s.DeleteOfflineMessages("ortuman"))
	s.DeactivateMockedError()
	require.Nil(t, s.DeleteOfflineMessages("ortuman"))

	elems, _ := s.FetchOfflineMessages("ortuman")
	require.Equal(t, 0, len(elems))
}
