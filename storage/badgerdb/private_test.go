/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_PrivateXML(t *testing.T) {
	t.Parallel()

	s, teardown := newPrivateMock()
	defer teardown()

	pv1 := xmpp.NewElementNamespace("ex1", "exodus:ns")
	pv2 := xmpp.NewElementNamespace("ex2", "exodus:ns")

	require.NoError(t, s.UpsertPrivateXML(context.Background(), []xmpp.XElement{pv1, pv2}, "exodus:ns", "ortuman"))

	pvs, err := s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman")
	require.Nil(t, err)
	require.Equal(t, 2, len(pvs))

	pvs2, err := s.FetchPrivateXML(context.Background(), "exodus:ns", "ortuman2")
	require.Nil(t, pvs2)
	require.Nil(t, err)
}

func newPrivateMock() (*badgerDBPrivate, func()) {
	t := newT()
	return &badgerDBPrivate{badgerDBStorage: newStorage(t.db)}, func() {
		t.teardown()
	}
}
