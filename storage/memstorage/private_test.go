/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertPrivateXML(t *testing.T) {
	private := xmpp.NewElementNamespace("exodus", "exodus:ns")

	s := New()
	s.EnableMockedError()
	err := s.InsertOrUpdatePrivateXML([]xmpp.XElement{private}, "exodus:ns", "ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	err = s.InsertOrUpdatePrivateXML([]xmpp.XElement{private}, "exodus:ns", "ortuman")
	require.Nil(t, err)
}

func TestMemoryStorage_FetchPrivateXML(t *testing.T) {
	private := xmpp.NewElementNamespace("exodus", "exodus:ns")

	s := New()
	_ = s.InsertOrUpdatePrivateXML([]xmpp.XElement{private}, "exodus:ns", "ortuman")

	s.EnableMockedError()
	_, err := s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	elems, _ := s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Equal(t, 1, len(elems))
}
