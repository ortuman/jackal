/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"testing"

	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_RosterItems(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	ri1 := &rostermodel.Item{
		Username:     "ortuman",
		JID:          "juliet",
		Subscription: "both",
	}
	ri2 := &rostermodel.Item{
		Username:     "ortuman",
		JID:          "romeo",
		Subscription: "both",
	}
	_, err := h.db.InsertOrUpdateRosterItem(ri1)
	require.NoError(t, err)
	_, err = h.db.InsertOrUpdateRosterItem(ri2)
	require.NoError(t, err)

	ris, _, err := h.db.FetchRosterItems("ortuman")
	require.Nil(t, err)
	require.Equal(t, 2, len(ris))

	ris2, _, err := h.db.FetchRosterItems("ortuman2")
	require.Nil(t, err)
	require.Equal(t, 0, len(ris2))

	ri3, err := h.db.FetchRosterItem("ortuman", "juliet")
	require.Nil(t, err)
	require.Equal(t, ri1, ri3)

	_, err = h.db.DeleteRosterItem("ortuman", "juliet")
	require.NoError(t, err)
	_, err = h.db.DeleteRosterItem("ortuman", "romeo")
	require.NoError(t, err)

	ris, _, err = h.db.FetchRosterItems("ortuman")
	require.Nil(t, err)
	require.Equal(t, 0, len(ris))
}

func TestBadgerDB_RosterNotifications(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	rn1 := rostermodel.Notification{
		Contact:  "ortuman",
		JID:      "juliet@jackal.im",
		Presence: &xmpp.Presence{},
	}
	rn2 := rostermodel.Notification{
		Contact:  "ortuman",
		JID:      "romeo@jackal.im",
		Presence: &xmpp.Presence{},
	}
	require.NoError(t, h.db.InsertOrUpdateRosterNotification(&rn1))
	require.NoError(t, h.db.InsertOrUpdateRosterNotification(&rn2))

	rns, err := h.db.FetchRosterNotifications("ortuman")
	require.Nil(t, err)
	require.Equal(t, 2, len(rns))

	rns2, err := h.db.FetchRosterNotifications("ortuman2")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns2))

	require.NoError(t, h.db.DeleteRosterNotification(rn1.Contact, rn1.JID))

	rns, err = h.db.FetchRosterNotifications("ortuman")
	require.Nil(t, err)
	require.Equal(t, 1, len(rns))

	require.NoError(t, h.db.DeleteRosterNotification(rn2.Contact, rn2.JID))

	rns, err = h.db.FetchRosterNotifications("ortuman")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))
}
