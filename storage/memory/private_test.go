/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memorystorage

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertPrivateXML(t *testing.T) {
	private := xmpp.NewElementNamespace("exodus", "exodus:ns")

	s := NewPrivate()
	EnableMockedError()
	err := s.UpsertPrivateXML(context.Background(), []xmpp.XElement{private}, "exodus:ns", "ortuman")
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	err = s.UpsertPrivateXML(context.Background(), []xmpp.XElement{private}, "exodus:ns", "ortuman")
	require.Nil(t, err)
}

func TestMemoryStorage_FetchPrivateXML(t *testing.T) {
	private := xmpp.NewElementNamespace("exodus", "exodus:ns")

	s := NewPrivate()
	_ = s.UpsertPrivateXML(context.Background(), []xmpp.XElement{private}, "exodus:ns", "ortuman")

	EnableMockedError()
	_, err := s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman")
	require.Equal(t, ErrMocked, err)
	DisableMockedError()

	elems, _ := s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman")
	require.Equal(t, 1, len(elems))
}
