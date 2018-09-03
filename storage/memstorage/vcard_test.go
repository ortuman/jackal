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

func TestMockStorageInsertVCard(t *testing.T) {
	vCard := xmpp.NewElementName("vCard")
	fn := xmpp.NewElementName("FN")
	fn.SetText("Miguel Ángel")
	vCard.AppendElement(fn)

	s := New()
	s.ActivateMockedError()
	require.Equal(t, ErrMockedError, s.InsertOrUpdateVCard(vCard, "ortuman"))
	s.DeactivateMockedError()
	require.Nil(t, s.InsertOrUpdateVCard(vCard, "ortuman"))
}

func TestMockStorageFetchVCard(t *testing.T) {
	vCard := xmpp.NewElementName("vCard")
	fn := xmpp.NewElementName("FN")
	fn.SetText("Miguel Ángel")
	vCard.AppendElement(fn)

	s := New()
	s.InsertOrUpdateVCard(vCard, "ortuman")

	s.ActivateMockedError()
	_, err := s.FetchVCard("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DeactivateMockedError()
	elem, _ := s.FetchVCard("ortuman")
	require.NotNil(t, elem)
}
