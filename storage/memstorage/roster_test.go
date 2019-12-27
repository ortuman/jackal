/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"context"
	"testing"

	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_InsertRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := rostermodel.Item{
		Username:     "user",
		JID:          "contact",
		Name:         "a name",
		Subscription: "both",
		Ask:          false,
		Ver:          1,
		Groups:       g,
	}

	s := New()
	s.EnableMockedError()
	_, err := s.UpsertRosterItem(context.Background(), &ri)
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	_, err = s.UpsertRosterItem(context.Background(), &ri)
	require.Nil(t, err)
	ri.Subscription = "to"
	_, err = s.UpsertRosterItem(context.Background(), &ri)
	require.Nil(t, err)
}

func TestMemoryStorage_FetchRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := rostermodel.Item{
		Username:     "user",
		JID:          "contact",
		Name:         "a name",
		Subscription: "both",
		Ask:          false,
		Ver:          1,
		Groups:       g,
	}
	s := New()
	_, _ = s.UpsertRosterItem(context.Background(), &ri)

	s.EnableMockedError()
	_, err := s.FetchRosterItem(context.Background(), "user", "contact")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()

	ri3, _ := s.FetchRosterItem(context.Background(), "user", "contact2")
	require.Nil(t, ri3)

	ri4, _ := s.FetchRosterItem(context.Background(), "user", "contact")
	require.NotNil(t, ri4)
	require.Equal(t, "user", ri4.Username)
	require.Equal(t, "contact", ri4.JID)
}

func TestMemoryStorage_FetchRosterItems(t *testing.T) {
	ri := rostermodel.Item{
		Username:     "user",
		JID:          "contact@jackal.im",
		Name:         "a name",
		Subscription: "both",
		Ask:          false,
		Ver:          1,
		Groups:       []string{"general", "friends"},
	}
	ri2 := rostermodel.Item{
		Username:     "user",
		JID:          "contact2@jackal.im",
		Name:         "a name 2",
		Subscription: "both",
		Ask:          false,
		Ver:          2,
		Groups:       []string{"general", "buddies"},
	}
	ri3 := rostermodel.Item{
		Username:     "user",
		JID:          "contact3@jackal.im",
		Name:         "a name 3",
		Subscription: "both",
		Ask:          false,
		Ver:          2,
		Groups:       []string{"family", "friends"},
	}

	s := New()
	_, _ = s.UpsertRosterItem(context.Background(), &ri)
	_, _ = s.UpsertRosterItem(context.Background(), &ri2)
	_, _ = s.UpsertRosterItem(context.Background(), &ri3)

	s.EnableMockedError()
	_, _, err := s.FetchRosterItems(context.Background(), "user")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()

	ris, _, _ := s.FetchRosterItems(context.Background(), "user")
	require.Equal(t, 3, len(ris))
	ris, _, _ = s.FetchRosterItemsInGroups(context.Background(), "user", []string{"friends"})
	require.Equal(t, 2, len(ris))
	ris, _, _ = s.FetchRosterItemsInGroups(context.Background(), "user", []string{"buddies"})
	require.Equal(t, 1, len(ris))

	gr, err := s.FetchRosterGroups(context.Background(), "user")
	require.Len(t, gr, 4)

	require.Contains(t, gr, "general")
	require.Contains(t, gr, "friends")
	require.Contains(t, gr, "family")
	require.Contains(t, gr, "buddies")
}

func TestMemoryStorage_DeleteRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := rostermodel.Item{
		Username:     "user",
		JID:          "contact",
		Name:         "a name",
		Subscription: "both",
		Ask:          false,
		Ver:          1,
		Groups:       g,
	}
	s := New()
	_, _ = s.UpsertRosterItem(context.Background(), &ri)

	gr, err := s.FetchRosterGroups(context.Background(), "user")
	require.Len(t, gr, 2)

	require.Contains(t, gr, "general")
	require.Contains(t, gr, "friends")

	s.EnableMockedError()
	_, err = s.DeleteRosterItem(context.Background(), "user", "contact")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()

	_, err = s.DeleteRosterItem(context.Background(), "user", "contact")
	require.Nil(t, err)
	_, err = s.DeleteRosterItem(context.Background(), "user2", "contact")
	require.Nil(t, err) // delete not existing roster item...

	ri2, _ := s.FetchRosterItem(context.Background(), "user", "contact")
	require.Nil(t, ri2)

	gr, err = s.FetchRosterGroups(context.Background(), "user")
	require.Len(t, gr, 0)
}

func TestMemoryStorage_InsertRosterNotification(t *testing.T) {
	rn := rostermodel.Notification{
		Contact:  "ortuman",
		JID:      "romeo@jackal.im",
		Presence: &xmpp.Presence{},
	}
	s := New()
	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.UpsertRosterNotification(context.Background(), &rn))
	s.DisableMockedError()
	require.Nil(t, s.UpsertRosterNotification(context.Background(), &rn))
}

func TestMemoryStorage_FetchRosterNotifications(t *testing.T) {
	rn1 := rostermodel.Notification{
		Contact:  "romeo",
		JID:      "ortuman@jackal.im",
		Presence: &xmpp.Presence{},
	}
	rn2 := rostermodel.Notification{
		Contact:  "romeo",
		JID:      "ortuman2@jackal.im",
		Presence: &xmpp.Presence{},
	}
	s := New()
	_ = s.UpsertRosterNotification(context.Background(), &rn1)
	_ = s.UpsertRosterNotification(context.Background(), &rn2)

	from, _ := jid.NewWithString("ortuman2@jackal.im", true)
	to, _ := jid.NewWithString("romeo@jackal.im", true)
	rn2.Presence = xmpp.NewPresence(from, to, xmpp.SubscribeType)
	_ = s.UpsertRosterNotification(context.Background(), &rn2)

	s.EnableMockedError()
	_, err := s.FetchRosterNotifications(context.Background(), "romeo")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	rns, err := s.FetchRosterNotifications(context.Background(), "romeo")
	require.Nil(t, err)
	require.Equal(t, 2, len(rns))
	require.Equal(t, "ortuman@jackal.im", rns[0].JID)
	require.Equal(t, "ortuman2@jackal.im", rns[1].JID)
}

func TestMemoryStorage_DeleteRosterNotification(t *testing.T) {
	rn1 := rostermodel.Notification{
		Contact:  "ortuman",
		JID:      "romeo@jackal.im",
		Presence: &xmpp.Presence{},
	}
	s := New()
	_ = s.UpsertRosterNotification(context.Background(), &rn1)

	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.DeleteRosterNotification(context.Background(), "ortuman", "romeo@jackal.im"))
	s.DisableMockedError()
	require.Nil(t, s.DeleteRosterNotification(context.Background(), "ortuman", "romeo@jackal.im"))

	rns, err := s.FetchRosterNotifications(context.Background(), "romeo")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))

	// delete not existing roster notification...
	require.Nil(t, s.DeleteRosterNotification(context.Background(), "ortuman2", "romeo@jackal.im"))
}
