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

func TestBadgerDB_VCard(t *testing.T) {
	t.Parallel()

	s, teardown := newVCardMock()
	defer teardown()

	vCard := xmpp.NewElementNamespace("vCard", "vcard-temp")
	fn := xmpp.NewElementName("FN")
	fn.SetText("Miguel Ángel Ortuño")
	vCard.AppendElement(fn)

	err := s.UpsertVCard(context.Background(), vCard, "ortuman")
	require.Nil(t, err)

	vCard2, err := s.FetchVCard(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, "vCard", vCard2.Name())
	require.Equal(t, "vcard-temp", vCard2.Namespace())
	require.NotNil(t, vCard2.Elements().Child("FN"))

	vCard3, err := s.FetchVCard(context.Background(), "ortuman2")
	require.Nil(t, vCard3)
	require.Nil(t, err)
}

func newVCardMock() (*badgerDBVCard, func()) {
	t := newT()
	return &badgerDBVCard{badgerDBStorage: newStorage(t.db)}, func() {
		t.teardown()
	}
}
