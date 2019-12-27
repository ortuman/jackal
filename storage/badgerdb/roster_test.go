/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package badgerdb

import (
	"context"
	"testing"

	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestBadgerDB_RosterItems(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	ri1 := &rostermodel.Item{
		Username:     "ortuman",
		JID:          "juliet@jackal.im",
		Subscription: "both",
		Groups:       []string{"general", "friends"},
	}
	ri2 := &rostermodel.Item{
		Username:     "ortuman",
		JID:          "romeo@jackal.im",
		Subscription: "both",
		Groups:       []string{"general", "buddies"},
	}
	ri3 := &rostermodel.Item{
		Username:     "ortuman",
		JID:          "hamlet@jackal.im",
		Subscription: "both",
		Groups:       []string{"family", "friends"},
	}
	_, err := h.db.UpsertRosterItem(context.Background(), ri1)
	require.Nil(t, err)
	_, err = h.db.UpsertRosterItem(context.Background(), ri2)
	require.Nil(t, err)
	_, err = h.db.UpsertRosterItem(context.Background(), ri3)
	require.Nil(t, err)

	ris, _, err := h.db.FetchRosterItems(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, 3, len(ris))

	ris, _, err = h.db.FetchRosterItemsInGroups(context.Background(), "ortuman", []string{"friends"})
	require.Nil(t, err)
	require.Equal(t, 2, len(ris))

	ris, _, err = h.db.FetchRosterItemsInGroups(context.Background(), "ortuman", []string{"general"})
	require.Nil(t, err)
	require.Equal(t, 2, len(ris))

	ris, _, err = h.db.FetchRosterItemsInGroups(context.Background(), "ortuman", []string{"buddies"})
	require.Nil(t, err)
	require.Equal(t, 1, len(ris))

	ris2, _, err := h.db.FetchRosterItems(context.Background(), "ortuman2")
	require.Nil(t, err)
	require.Equal(t, 0, len(ris2))

	ri4, err := h.db.FetchRosterItem(context.Background(), "ortuman", "juliet@jackal.im")
	require.Nil(t, err)
	require.Equal(t, ri1, ri4)

	gr, err := h.db.FetchRosterGroups(context.Background(), "ortuman")
	require.Len(t, gr, 4)

	require.Contains(t, gr, "general")
	require.Contains(t, gr, "friends")
	require.Contains(t, gr, "family")
	require.Contains(t, gr, "buddies")

	_, err = h.db.DeleteRosterItem(context.Background(), "ortuman", "juliet@jackal.im")
	require.NoError(t, err)
	_, err = h.db.DeleteRosterItem(context.Background(), "ortuman", "romeo@jackal.im")
	require.NoError(t, err)
	_, err = h.db.DeleteRosterItem(context.Background(), "ortuman", "hamlet@jackal.im")
	require.NoError(t, err)

	ris, _, err = h.db.FetchRosterItems(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, 0, len(ris))

	gr, err = h.db.FetchRosterGroups(context.Background(), "ortuman")
	require.Len(t, gr, 0)
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
	require.NoError(t, h.db.UpsertRosterNotification(context.Background(), &rn1))
	require.NoError(t, h.db.UpsertRosterNotification(context.Background(), &rn2))

	rns, err := h.db.FetchRosterNotifications(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, 2, len(rns))

	rns2, err := h.db.FetchRosterNotifications(context.Background(), "ortuman2")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns2))

	require.NoError(t, h.db.DeleteRosterNotification(context.Background(), rn1.Contact, rn1.JID))

	rns, err = h.db.FetchRosterNotifications(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, 1, len(rns))

	require.NoError(t, h.db.DeleteRosterNotification(context.Background(), rn2.Contact, rn2.JID))

	rns, err = h.db.FetchRosterNotifications(context.Background(), "ortuman")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))
}
