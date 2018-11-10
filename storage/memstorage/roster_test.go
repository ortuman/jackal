/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"testing"

	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestMockStorageInsertRosterItem(t *testing.T) {
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
	_, err := s.InsertOrUpdateRosterItem(&ri)
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	_, err = s.InsertOrUpdateRosterItem(&ri)
	require.Nil(t, err)
	ri.Subscription = "to"
	_, err = s.InsertOrUpdateRosterItem(&ri)
	require.Nil(t, err)
}

func TestMockStorageFetchRosterItem(t *testing.T) {
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
	s.InsertOrUpdateRosterItem(&ri)

	s.EnableMockedError()
	_, err := s.FetchRosterItem("user", "contact")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()

	ri3, _ := s.FetchRosterItem("user", "contact2")
	require.Nil(t, ri3)

	ri4, _ := s.FetchRosterItem("user", "contact")
	require.NotNil(t, ri4)
	require.Equal(t, "user", ri4.Username)
	require.Equal(t, "contact", ri4.JID)
}

func TestMockStorageFetchRosterItems(t *testing.T) {
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
	ri2 := rostermodel.Item{
		Username:     "user",
		JID:          "contact2",
		Name:         "a name 2",
		Subscription: "both",
		Ask:          false,
		Ver:          2,
		Groups:       g,
	}

	s := New()
	s.InsertOrUpdateRosterItem(&ri)
	s.InsertOrUpdateRosterItem(&ri2)

	s.EnableMockedError()
	_, _, err := s.FetchRosterItems("user")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	ris, _, _ := s.FetchRosterItems("user")
	require.Equal(t, 2, len(ris))
}

func TestMockStorageDeleteRosterItem(t *testing.T) {
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
	s.InsertOrUpdateRosterItem(&ri)

	s.EnableMockedError()
	_, err := s.DeleteRosterItem("user", "contact")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	_, err = s.DeleteRosterItem("user", "contact")
	require.Nil(t, err)
	_, err = s.DeleteRosterItem("user2", "contact")
	require.Nil(t, err) // delete not existing roster item...

	ri2, _ := s.FetchRosterItem("user", "contact")
	require.Nil(t, ri2)
}

func TestMockStorageInsertRosterNotification(t *testing.T) {
	rn := rostermodel.Notification{
		Contact:  "ortuman",
		JID:      "romeo@jackal.im",
		Presence: &xmpp.Presence{},
	}
	s := New()
	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.InsertOrUpdateRosterNotification(&rn))
	s.DisableMockedError()
	require.Nil(t, s.InsertOrUpdateRosterNotification(&rn))
}

func TestMockStorageFetchRosterNotifications(t *testing.T) {
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
	s.InsertOrUpdateRosterNotification(&rn1)
	s.InsertOrUpdateRosterNotification(&rn2)

	from, _ := jid.NewWithString("ortuman2@jackal.im", true)
	to, _ := jid.NewWithString("romeo@jackal.im", true)
	rn2.Presence = xmpp.NewPresence(from, to, xmpp.SubscribeType)
	s.InsertOrUpdateRosterNotification(&rn2)

	s.EnableMockedError()
	_, err := s.FetchRosterNotifications("romeo")
	require.Equal(t, ErrMockedError, err)
	s.DisableMockedError()
	rns, err := s.FetchRosterNotifications("romeo")
	require.Nil(t, err)
	require.Equal(t, 2, len(rns))
	require.Equal(t, "ortuman@jackal.im", rns[0].JID)
	require.Equal(t, "ortuman2@jackal.im", rns[1].JID)
}

func TestMockStorageDeleteRosterNotification(t *testing.T) {
	rn1 := rostermodel.Notification{
		Contact:  "ortuman",
		JID:      "romeo@jackal.im",
		Presence: &xmpp.Presence{},
	}
	s := New()
	s.InsertOrUpdateRosterNotification(&rn1)

	s.EnableMockedError()
	require.Equal(t, ErrMockedError, s.DeleteRosterNotification("ortuman", "romeo@jackal.im"))
	s.DisableMockedError()
	require.Nil(t, s.DeleteRosterNotification("ortuman", "romeo@jackal.im"))

	rns, err := s.FetchRosterNotifications("romeo")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))
	// delete not existing roster notification...
	require.Nil(t, s.DeleteRosterNotification("ortuman2", "romeo@jackal.im"))
}
