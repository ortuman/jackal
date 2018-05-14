/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_VCard(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	vcard := xml.NewElementNamespace("vCard", "vcard-temp")
	fn := xml.NewElementName("FN")
	fn.SetText("Miguel Ángel Ortuño")
	vcard.AppendElement(fn)

	err := h.db.InsertOrUpdateVCard(vcard, "ortuman")
	require.Nil(t, err)

	vcard2, err := h.db.FetchVCard("ortuman")
	require.Nil(t, err)
	require.Equal(t, "vCard", vcard2.Name())
	require.Equal(t, "vcard-temp", vcard2.Namespace())
	require.NotNil(t, vcard2.Elements().Child("FN"))

	vcard3, err := h.db.FetchVCard("ortuman2")
	require.Nil(t, vcard3)
	require.Nil(t, err)
}
