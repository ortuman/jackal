/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_OfflineMessages(t *testing.T) {
	t.Parallel()

	s, teardown := newOfflineMock()
	defer teardown()

	msg1 := xmpp.NewMessageType(uuid.New(), xmpp.NormalType)
	b1 := xmpp.NewElementName("body")
	b1.SetText("Hi buddy!")
	msg1.AppendElement(b1)

	msg2 := xmpp.NewMessageType(uuid.New(), xmpp.NormalType)
	b2 := xmpp.NewElementName("body")
	b2.SetText("what's up?!")
	msg1.AppendElement(b1)

	require.NoError(t, s.InsertOfflineMessage(context.Background(), msg1, "ortuman"))
	require.NoError(t, s.InsertOfflineMessage(context.Background(), msg2, "ortuman"))

	cnt, err := s.CountOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, 2, cnt)

	msgs, err := s.FetchOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, 2, len(msgs))

	msgs2, err := s.FetchOfflineMessages(context.Background(), "ortuman2")
	require.Nil(t, err)
	require.Equal(t, 0, len(msgs2))

	require.NoError(t, s.DeleteOfflineMessages(context.Background(), "ortuman"))
	cnt, err = s.CountOfflineMessages(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, 0, cnt)
}

func newOfflineMock() (*badgerDBOffline, func()) {
	t := newT()
	return &badgerDBOffline{badgerDBStorage: newStorage(t.DB)}, func() {
		t.teardown()
	}
}
