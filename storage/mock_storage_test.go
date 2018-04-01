/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"testing"

	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestMockStorageInsertUser(t *testing.T) {
	u := model.User{Username: "ortuman", Password: "1234"}
	s := newMockStorage()
	s.activateMockedError()
	err := s.InsertOrUpdateUser(&u)
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()
	err = s.InsertOrUpdateUser(&u)
	require.Nil(t, err)
}

func TestMockStorageUserExists(t *testing.T) {
	s := newMockStorage()
	s.activateMockedError()
	ok, err := s.UserExists("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()
	ok, err = s.UserExists("ortuman")
	require.Nil(t, err)
	require.False(t, ok)
}

func TestMockStorageFetchUser(t *testing.T) {
	u := model.User{Username: "ortuman", Password: "1234"}
	s := newMockStorage()
	_ = s.InsertOrUpdateUser(&u)

	s.activateMockedError()
	_, err := s.FetchUser("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()
	usr, _ := s.FetchUser("romeo")
	require.Nil(t, usr)
	usr, _ = s.FetchUser("ortuman")
	require.NotNil(t, usr)
}

func TestMockStorageDeleteUser(t *testing.T) {
	u := model.User{Username: "ortuman", Password: "1234"}
	s := newMockStorage()
	_ = s.InsertOrUpdateUser(&u)

	s.activateMockedError()
	require.Equal(t, ErrMockedError, s.DeleteUser("ortuman"))
	s.deactivateMockedError()
	require.Nil(t, s.DeleteUser("ortuman"))

	usr, _ := s.FetchUser("ortuman")
	require.Nil(t, usr)
}

func TestMockStorageInsertRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := model.RosterItem{"user", "contact", "a name", "both", false, g}

	s := newMockStorage()
	s.activateMockedError()
	require.Equal(t, ErrMockedError, s.InsertOrUpdateRosterItem(&ri))
	s.deactivateMockedError()
	require.Nil(t, s.InsertOrUpdateRosterItem(&ri))
	ri.Subscription = "to"
	require.Nil(t, s.InsertOrUpdateRosterItem(&ri))
}

func TestMockStorageFetchRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := model.RosterItem{"user", "contact", "a name", "both", false, g}

	s := newMockStorage()
	s.InsertOrUpdateRosterItem(&ri)

	s.activateMockedError()
	_, err := s.FetchRosterItem("user", "contact")
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()

	ri3, _ := s.FetchRosterItem("user", "contact2")
	require.Nil(t, ri3)

	ri4, _ := s.FetchRosterItem("user", "contact")
	require.NotNil(t, ri4)
	require.Equal(t, "user", ri4.User)
	require.Equal(t, "contact", ri4.Contact)
}

func TestMockStorageFetchRosterItems(t *testing.T) {
	g := []string{"general", "friends"}
	ri := model.RosterItem{"user", "contact", "a name", "both", false, g}
	ri2 := model.RosterItem{"user", "contact2", "a name 2", "both", false, g}

	s := newMockStorage()
	s.InsertOrUpdateRosterItem(&ri)
	s.InsertOrUpdateRosterItem(&ri2)

	s.activateMockedError()
	_, err := s.FetchRosterItems("user")
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()
	ris, _ := s.FetchRosterItems("user")
	require.Equal(t, 2, len(ris))
}

func TestMockStorageDeleteRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := model.RosterItem{"user", "contact", "a name", "both", false, g}
	s := newMockStorage()
	s.InsertOrUpdateRosterItem(&ri)

	s.activateMockedError()
	require.Equal(t, ErrMockedError, s.DeleteRosterItem("user", "contact"))
	s.deactivateMockedError()
	require.Nil(t, s.DeleteRosterItem("user", "contact"))
	require.Nil(t, s.DeleteRosterItem("user2", "contact")) // delete not existing roster item...

	ri2, _ := s.FetchRosterItem("user", "contact")
	require.Nil(t, ri2)
}

func TestMockStorageInsertRosterNotification(t *testing.T) {
	rn := model.RosterNotification{
		"ortuman",
		"romeo",
		[]xml.XElement{xml.NewElementName("priority")},
	}
	s := newMockStorage()
	s.activateMockedError()
	require.Equal(t, ErrMockedError, s.InsertOrUpdateRosterNotification(&rn))
	s.deactivateMockedError()
	require.Nil(t, s.InsertOrUpdateRosterNotification(&rn))
}

func TestMockStorageFetchRosterNotifications(t *testing.T) {
	rn1 := model.RosterNotification{
		"ortuman",
		"romeo",
		[]xml.XElement{xml.NewElementName("priority")},
	}
	rn2 := model.RosterNotification{
		"ortuman2",
		"romeo",
		[]xml.XElement{xml.NewElementName("priority")},
	}
	s := newMockStorage()
	s.InsertOrUpdateRosterNotification(&rn1)
	s.InsertOrUpdateRosterNotification(&rn2)

	rn2.Elements = []xml.XElement{xml.NewElementName("status")}
	s.InsertOrUpdateRosterNotification(&rn2)

	s.activateMockedError()
	_, err := s.FetchRosterNotifications("romeo")
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()
	rns, err := s.FetchRosterNotifications("romeo")
	require.Nil(t, err)
	require.Equal(t, 2, len(rns))
	require.Equal(t, "ortuman", rns[0].User)
	require.Equal(t, "ortuman2", rns[1].User)
}

func TestMockStorageDeleteRosterNotification(t *testing.T) {
	rn1 := model.RosterNotification{
		"ortuman",
		"romeo",
		[]xml.XElement{xml.NewElementName("priority")},
	}
	s := newMockStorage()
	s.InsertOrUpdateRosterNotification(&rn1)

	s.activateMockedError()
	require.Equal(t, ErrMockedError, s.DeleteRosterNotification("ortuman", "romeo"))
	s.deactivateMockedError()
	require.Nil(t, s.DeleteRosterNotification("ortuman", "romeo"))

	rns, err := s.FetchRosterNotifications("romeo")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))
	// delete not existing roster notification...
	require.Nil(t, s.DeleteRosterNotification("ortuman2", "romeo"))
}

func TestMockStorageInsertVCard(t *testing.T) {
	vCard := xml.NewElementName("vCard")
	fn := xml.NewElementName("FN")
	fn.SetText("Miguel Ángel")
	vCard.AppendElement(fn)

	s := newMockStorage()
	s.activateMockedError()
	require.Equal(t, ErrMockedError, s.InsertOrUpdateVCard(vCard, "ortuman"))
	s.deactivateMockedError()
	require.Nil(t, s.InsertOrUpdateVCard(vCard, "ortuman"))
}

func TestMockStorageFetchVCard(t *testing.T) {
	vCard := xml.NewElementName("vCard")
	fn := xml.NewElementName("FN")
	fn.SetText("Miguel Ángel")
	vCard.AppendElement(fn)

	s := newMockStorage()
	s.InsertOrUpdateVCard(vCard, "ortuman")

	s.activateMockedError()
	_, err := s.FetchVCard("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()
	elem, _ := s.FetchVCard("ortuman")
	require.NotNil(t, elem)
}

func TestMockStorageInsertPrivateXML(t *testing.T) {
	private := xml.NewElementNamespace("exodus", "exodus:ns")

	s := newMockStorage()
	s.activateMockedError()
	err := s.InsertOrUpdatePrivateXML([]xml.XElement{private}, "exodus:ns", "ortuman")
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()
	err = s.InsertOrUpdatePrivateXML([]xml.XElement{private}, "exodus:ns", "ortuman")
	require.Nil(t, err)
}

func TestMockStorageFetchPrivateXML(t *testing.T) {
	private := xml.NewElementNamespace("exodus", "exodus:ns")

	s := newMockStorage()
	s.InsertOrUpdatePrivateXML([]xml.XElement{private}, "exodus:ns", "ortuman")

	s.activateMockedError()
	_, err := s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()
	elems, _ := s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Equal(t, 1, len(elems))
}

func TestMockStorageInsertOfflineMessage(t *testing.T) {
	j, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	message := xml.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xml.NewElementName("body"))
	m, _ := xml.NewMessageFromElement(message, j, j)

	s := newMockStorage()
	s.activateMockedError()
	require.Equal(t, ErrMockedError, s.InsertOfflineMessage(m, "ortuman"))
	s.deactivateMockedError()
	require.Nil(t, s.InsertOfflineMessage(m, "ortuman"))
}

func TestMockStorageCountOfflineMessages(t *testing.T) {
	j, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	message := xml.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xml.NewElementName("body"))
	m, _ := xml.NewMessageFromElement(message, j, j)

	s := newMockStorage()
	s.InsertOfflineMessage(m, "ortuman")

	s.activateMockedError()
	_, err := s.CountOfflineMessages("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()
	cnt, _ := s.CountOfflineMessages("ortuman")
	require.Equal(t, 1, cnt)
}

func TestMockStorageFetchOfflineMessages(t *testing.T) {
	j, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	message := xml.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xml.NewElementName("body"))
	m, _ := xml.NewMessageFromElement(message, j, j)

	s := newMockStorage()
	s.InsertOfflineMessage(m, "ortuman")

	s.activateMockedError()
	_, err := s.FetchOfflineMessages("ortuman")
	require.Equal(t, ErrMockedError, err)
	s.deactivateMockedError()
	elems, _ := s.FetchOfflineMessages("ortuman")
	require.Equal(t, 1, len(elems))
}

func TestMockStorageDeleteOfflineMessages(t *testing.T) {
	j, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	message := xml.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xml.NewElementName("body"))
	m, _ := xml.NewMessageFromElement(message, j, j)

	s := newMockStorage()
	s.InsertOfflineMessage(m, "ortuman")

	s.activateMockedError()
	require.Equal(t, ErrMockedError, s.DeleteOfflineMessages("ortuman"))
	s.deactivateMockedError()
	require.Nil(t, s.DeleteOfflineMessages("ortuman"))

	elems, _ := s.FetchOfflineMessages("ortuman")
	require.Equal(t, 0, len(elems))
}
