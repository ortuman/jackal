/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memory

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertPrivateXML(t *testing.T) {
	private := xmpp.NewElementNamespace("exodus", "exodus:ns")

	s := New2()
	s.EnableMockedError()
	err := s.UpsertPrivateXML(context.Background(), []xmpp.XElement{private}, "exodus:ns", "ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	err = s.UpsertPrivateXML(context.Background(), []xmpp.XElement{private}, "exodus:ns", "ortuman")
	require.Nil(t, err)
}

func TestMemoryStorage_FetchPrivateXML(t *testing.T) {
	private := xmpp.NewElementNamespace("exodus", "exodus:ns")

	s := New2()
	_ = s.UpsertPrivateXML(context.Background(), []xmpp.XElement{private}, "exodus:ns", "ortuman")

	s.EnableMockedError()
	_, err := s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	elems, _ := s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman")
	require.Equal(t, 1, len(elems))
}
