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

func TestBadgerDB_PrivateXML(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	pv1 := xml.NewElementNamespace("ex1", "exodus:ns")
	pv2 := xml.NewElementNamespace("ex2", "exodus:ns")

	require.NoError(t, h.db.InsertOrUpdatePrivateXML([]xml.XElement{pv1, pv2}, "exodus:ns", "ortuman"))

	prvs, err := h.db.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, err)
	require.Equal(t, 2, len(prvs))

	prvs2, err := h.db.FetchPrivateXML("exodus:ns", "ortuman2")
	require.Nil(t, prvs2)
	require.Nil(t, err)
}
