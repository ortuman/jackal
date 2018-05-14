/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_OfflineMessages(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	msg1 := xml.NewMessageType(uuid.New(), xml.NormalType)
	b1 := xml.NewElementName("body")
	b1.SetText("Hi buddy!")
	msg1.AppendElement(b1)

	msg2 := xml.NewMessageType(uuid.New(), xml.NormalType)
	b2 := xml.NewElementName("body")
	b2.SetText("what's up?!")
	msg1.AppendElement(b1)

	require.NoError(t, h.db.InsertOfflineMessage(msg1, "ortuman"))
	require.NoError(t, h.db.InsertOfflineMessage(msg2, "ortuman"))

	cnt, err := h.db.CountOfflineMessages("ortuman")
	require.Nil(t, err)
	require.Equal(t, 2, cnt)

	msgs, err := h.db.FetchOfflineMessages("ortuman")
	require.Nil(t, err)
	require.Equal(t, 2, len(msgs))

	msgs2, err := h.db.FetchOfflineMessages("ortuman2")
	require.Nil(t, err)
	require.Equal(t, 0, len(msgs2))

	require.NoError(t, h.db.DeleteOfflineMessages("ortuman"))
	cnt, err = h.db.CountOfflineMessages("ortuman")
	require.Nil(t, err)
	require.Equal(t, 0, cnt)
}
