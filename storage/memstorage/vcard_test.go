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

func TestMemoryStorage_InsertVCard(t *testing.T) {
	vCard := xmpp.NewElementName("vCard")
	fn := xmpp.NewElementName("FN")
	fn.SetText("Miguel Ángel")
	vCard.AppendElement(fn)

	s := New()
	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.InsertOrUpdateVCard(vCard, "ortuman"))
	s.DisableMockedError()
	require.Nil(t, s.InsertOrUpdateVCard(vCard, "ortuman"))
}

func TestMemoryStorage_FetchVCard(t *testing.T) {
	vCard := xmpp.NewElementName("vCard")
	fn := xmpp.NewElementName("FN")
	fn.SetText("Miguel Ángel")
	vCard.AppendElement(fn)

	s := New()
	s.InsertOrUpdateVCard(vCard, "ortuman")

	s.EnableMockedError()
	_, err := s.FetchVCard("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	elem, _ := s.FetchVCard("ortuman")
	require.NotNil(t, elem)
}
