/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertVCard(t *testing.T) {
	vCard := xmpp.NewElementName("vCard")
	fn := xmpp.NewElementName("FN")
	fn.SetText("Miguel Ángel")
	vCard.AppendElement(fn)

	s := New()
	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.UpsertVCard(context.Background(), vCard, "ortuman"))
	s.DisableMockedError()
	require.Nil(t, s.UpsertVCard(context.Background(), vCard, "ortuman"))
}

func TestMemoryStorage_FetchVCard(t *testing.T) {
	vCard := xmpp.NewElementName("vCard")
	fn := xmpp.NewElementName("FN")
	fn.SetText("Miguel Ángel")
	vCard.AppendElement(fn)

	s := New()
	_ = s.UpsertVCard(context.Background(), vCard, "ortuman")

	s.EnableMockedError()
	_, err := s.FetchVCard(context.Background(), "ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()

	elem, _ := s.FetchVCard(context.Background(), "ortuman")
	require.NotNil(t, elem)
}
