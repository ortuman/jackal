/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package memstorage

import (
	"testing"

	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

func TestMockStorageInsertRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := model.RosterItem{"user", "contact", "a name", "both", false, 1, g}

	s := New()
	s.ActivateMockedError()
	_, err := s.InsertOrUpdateRosterItem(&ri)
	require.Equal(t, ErrMockedError, err)
	s.DeactivateMockedError()
	_, err = s.InsertOrUpdateRosterItem(&ri)
	require.Nil(t, err)
	ri.Subscription = "to"
	_, err = s.InsertOrUpdateRosterItem(&ri)
	require.Nil(t, err)
}

func TestMockStorageFetchRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := model.RosterItem{"user", "contact", "a name", "both", false, 1, g}

	s := New()
	s.InsertOrUpdateRosterItem(&ri)

	s.ActivateMockedError()
	_, err := s.FetchRosterItem("user", "contact")
	require.Equal(t, ErrMockedError, err)
	s.DeactivateMockedError()

	ri3, _ := s.FetchRosterItem("user", "contact2")
	require.Nil(t, ri3)

	ri4, _ := s.FetchRosterItem("user", "contact")
	require.NotNil(t, ri4)
	require.Equal(t, "user", ri4.Username)
	require.Equal(t, "contact", ri4.JID)
}

func TestMockStorageFetchRosterItems(t *testing.T) {
	g := []string{"general", "friends"}
	ri := model.RosterItem{"user", "contact", "a name", "both", false, 1, g}
	ri2 := model.RosterItem{"user", "contact2", "a name 2", "both", false, 2, g}

	s := New()
	s.InsertOrUpdateRosterItem(&ri)
	s.InsertOrUpdateRosterItem(&ri2)

	s.ActivateMockedError()
	_, _, err := s.FetchRosterItems("user")
	require.Equal(t, ErrMockedError, err)
	s.DeactivateMockedError()
	ris, _, _ := s.FetchRosterItems("user")
	require.Equal(t, 2, len(ris))
}

func TestMockStorageDeleteRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := model.RosterItem{"user", "contact", "a name", "both", false, 1, g}
	s := New()
	s.InsertOrUpdateRosterItem(&ri)

	s.ActivateMockedError()
	_, err := s.DeleteRosterItem("user", "contact")
	require.Equal(t, ErrMockedError, err)
	s.DeactivateMockedError()
	_, err = s.DeleteRosterItem("user", "contact")
	require.Nil(t, err)
	_, err = s.DeleteRosterItem("user2", "contact")
	require.Nil(t, err) // delete not existing roster item...

	ri2, _ := s.FetchRosterItem("user", "contact")
	require.Nil(t, ri2)
}

func TestMockStorageInsertRosterNotification(t *testing.T) {
	rn := model.RosterNotification{
		"ortuman",
		"romeo",
		[]xml.XElement{xml.NewElementName("priority")},
	}
	s := New()
	s.ActivateMockedError()
	require.Equal(t, ErrMockedError, s.InsertOrUpdateRosterNotification(&rn))
	s.DeactivateMockedError()
	require.Nil(t, s.InsertOrUpdateRosterNotification(&rn))
}

func TestMockStorageFetchRosterNotifications(t *testing.T) {
	rn1 := model.RosterNotification{
		"romeo",
		"ortuman@jackal.im",
		[]xml.XElement{xml.NewElementName("priority")},
	}
	rn2 := model.RosterNotification{
		"romeo",
		"ortuman2@jackal.im",
		[]xml.XElement{xml.NewElementName("priority")},
	}
	s := New()
	s.InsertOrUpdateRosterNotification(&rn1)
	s.InsertOrUpdateRosterNotification(&rn2)

	rn2.Elements = []xml.XElement{xml.NewElementName("status")}
	s.InsertOrUpdateRosterNotification(&rn2)

	s.ActivateMockedError()
	_, err := s.FetchRosterNotifications("romeo")
	require.Equal(t, ErrMockedError, err)
	s.DeactivateMockedError()
	rns, err := s.FetchRosterNotifications("romeo")
	require.Nil(t, err)
	require.Equal(t, 2, len(rns))
	require.Equal(t, "ortuman@jackal.im", rns[0].JID)
	require.Equal(t, "ortuman2@jackal.im", rns[1].JID)
}

func TestMockStorageDeleteRosterNotification(t *testing.T) {
	rn1 := model.RosterNotification{
		"ortuman",
		"romeo",
		[]xml.XElement{xml.NewElementName("priority")},
	}
	s := New()
	s.InsertOrUpdateRosterNotification(&rn1)

	s.ActivateMockedError()
	require.Equal(t, ErrMockedError, s.DeleteRosterNotification("ortuman", "romeo"))
	s.DeactivateMockedError()
	require.Nil(t, s.DeleteRosterNotification("ortuman", "romeo"))

	rns, err := s.FetchRosterNotifications("romeo")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))
	// delete not existing roster notification...
	require.Nil(t, s.DeleteRosterNotification("ortuman2", "romeo"))
}
