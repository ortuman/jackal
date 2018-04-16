/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ortuman/jackal/pool"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

var (
	errMySQLStorage = errors.New("MySQL storage error")
)

func TestMySQLStorageInsertUser(t *testing.T) {
	now := time.Now()
	user := model.User{Username: "ortuman", Password: "1234", LoggedOutStatus: "Bye!", LoggedOutAt: now}

	s, mock := newMockMySQLStorage()
	mock.ExpectExec("INSERT INTO users (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "1234", "Bye!", "1234", "Bye!", now).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOrUpdateUser(&user)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectExec("INSERT INTO users (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "1234", "Bye!", "1234", "Bye!", now).
		WillReturnError(errMySQLStorage)
	err = s.InsertOrUpdateUser(&user)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageDeleteUser(t *testing.T) {
	s, mock := newMockMySQLStorage()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_items (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_versions (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM private_storage (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM vcards (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM users (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnError(errMySQLStorage)
	mock.ExpectRollback()

	err = s.DeleteUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchUser(t *testing.T) {
	var userColumns = []string{"username", "password", "logged_out_status", "logged_out_at"}

	s, mock := newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(userColumns))

	usr, err := s.FetchUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, usr)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(userColumns).AddRow("ortuman", "1234", "Bye!", time.Now()))
	_, err = s.FetchUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM users (.+)").
		WithArgs("ortuman").WillReturnError(errMySQLStorage)
	_, err = s.FetchUser("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageUserExists(t *testing.T) {
	countColums := []string{"count"}

	s, mock := newMockMySQLStorage()
	mock.ExpectQuery("SELECT COUNT(.+) FROM users (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(countColums).AddRow(1))

	ok, err := s.UserExists("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.True(t, ok)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT COUNT(.+) FROM users (.+)").
		WithArgs("romeo").
		WillReturnError(errMySQLStorage)
	_, err = s.UserExists("romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageInsertRosterItem(t *testing.T) {
	g := []string{"general", "friends"}
	ri := model.RosterItem{"user", "contact", "a name", "both", false, 1, g}

	args := []driver.Value{
		ri.User,
		ri.Contact,
		ri.Name,
		ri.Subscription,
		"general;friends",
		ri.Ask,
		ri.User,
		ri.Name,
		ri.Subscription,
		"general;friends",
		ri.Ask,
	}

	s, mock := newMockMySQLStorage()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO roster_versions (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("user").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO roster_items (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(1, 0))

	_, err := s.InsertOrUpdateRosterItem(&ri)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
}

func TestMySQLStorageDeleteRosterItem(t *testing.T) {
	s, mock := newMockMySQLStorage()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO roster_versions (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("user").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM roster_items (.+)").
		WithArgs("user", "contact").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(1, 0))

	_, err := s.DeleteRosterItem("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO roster_versions (.+)").
		WithArgs("user").WillReturnError(errMySQLStorage)
	mock.ExpectRollback()

	_, err = s.DeleteRosterItem("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchRosterItems(t *testing.T) {
	var riColumns = []string{"user", "contact", "name", "subscription", "groups", "ask", "ver"}

	s, mock := newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(riColumns).AddRow("ortuman", "romeo", "Romeo", "both", "", false, 0))
	mock.ExpectQuery("SELECT (.+) FROM roster_versions (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows([]string{"ver", "deletionVer"}).AddRow(0, 0))

	rosterItems, _, err := s.FetchRosterItems("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 1, len(rosterItems))

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	_, _, err = s.FetchRosterItems("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnRows(sqlmock.NewRows(riColumns).AddRow("ortuman", "romeo", "Romeo", "both", "", false, 0))

	ri, err := s.FetchRosterItem("ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnRows(sqlmock.NewRows(riColumns))

	ri, err = s.FetchRosterItem("ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, ri)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM roster_items (.+)").
		WithArgs("ortuman", "romeo").
		WillReturnError(errMySQLStorage)

	_, err = s.FetchRosterItem("ortuman", "romeo")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageInsertRosterNotification(t *testing.T) {
	rn := model.RosterNotification{
		"ortuman",
		"romeo",
		[]xml.XElement{xml.NewElementName("priority")},
	}
	p := pool.NewBufferPool()

	buf := p.Get()
	defer p.Put(buf)
	for _, elem := range rn.Elements {
		buf.WriteString(elem.String())
	}
	elementsXML := buf.String()

	args := []driver.Value{
		rn.User,
		rn.Contact,
		elementsXML,
		elementsXML,
	}
	s, mock := newMockMySQLStorage()
	mock.ExpectExec("INSERT INTO roster_notifications (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOrUpdateRosterNotification(&rn)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectExec("INSERT INTO roster_notifications (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs(args...).
		WillReturnError(errMySQLStorage)

	err = s.InsertOrUpdateRosterNotification(&rn)
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageDeleteRosterNotification(t *testing.T) {
	s, mock := newMockMySQLStorage()
	mock.ExpectExec("DELETE FROM roster_notifications (.+)").
		WithArgs("user", "contact").WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeleteRosterNotification("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectExec("DELETE FROM roster_notifications (.+)").
		WithArgs("user", "contact").WillReturnError(errMySQLStorage)

	err = s.DeleteRosterNotification("user", "contact")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchRosterNotifications(t *testing.T) {
	var rnColumns = []string{"user", "contact", "elements"}

	s, mock := newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(rnColumns).AddRow("romeo", "contact", "<priority>8</priority>"))

	rosterNotifications, err := s.FetchRosterNotifications("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 1, len(rosterNotifications))

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(rnColumns))

	rosterNotifications, err = s.FetchRosterNotifications("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 0, len(rosterNotifications))

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	_, err = s.FetchRosterNotifications("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM roster_notifications (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(rnColumns).AddRow("romeo", "contact", "<priority>8"))

	_, err = s.FetchRosterNotifications("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)
}

func TestMySQLStorageInsertVCard(t *testing.T) {
	vCard := xml.NewElementName("vCard")
	rawXML := vCard.String()

	s, mock := newMockMySQLStorage()
	mock.ExpectExec("INSERT INTO vcards (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", rawXML, rawXML).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOrUpdateVCard(vCard, "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.NotNil(t, vCard)

	s, mock = newMockMySQLStorage()
	mock.ExpectExec("INSERT INTO vcards (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", rawXML, rawXML).
		WillReturnError(errMySQLStorage)

	err = s.InsertOrUpdateVCard(vCard, "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchVCard(t *testing.T) {
	var vCardColumns = []string{"vcard"}

	s, mock := newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM vcards (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(vCardColumns).AddRow("<vCard><FN>Miguel Ángel</FN></vCard>"))

	vCard, err := s.FetchVCard("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.NotNil(t, vCard)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM vcards (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(vCardColumns))

	vCard, err = s.FetchVCard("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Nil(t, vCard)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM vcards (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	vCard, _ = s.FetchVCard("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, vCard)
}

func TestMySQLStorageInsertPrivateXML(t *testing.T) {
	private := xml.NewElementNamespace("exodus", "exodus:ns")
	rawXML := private.String()

	s, mock := newMockMySQLStorage()
	mock.ExpectExec("INSERT INTO private_storage (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "exodus:ns", rawXML, rawXML).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOrUpdatePrivateXML([]xml.XElement{private}, "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectExec("INSERT INTO private_storage (.+) ON DUPLICATE KEY UPDATE (.+)").
		WithArgs("ortuman", "exodus:ns", rawXML, rawXML).
		WillReturnError(errMySQLStorage)

	err = s.InsertOrUpdatePrivateXML([]xml.XElement{private}, "exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchPrivateXML(t *testing.T) {
	var privateColumns = []string{"data"}

	s, mock := newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns).AddRow("<exodus xmlns='exodus:ns'><stuff/></exodus>"))

	elems, err := s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 1, len(elems))

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns).AddRow("<exodus xmlns='exodus:ns'><stuff/>"))

	elems, err = s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)
	require.Equal(t, 0, len(elems))

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns).AddRow(""))

	elems, err = s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 0, len(elems))

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnRows(sqlmock.NewRows(privateColumns))

	elems, err = s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)
	require.Equal(t, 0, len(elems))

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM private_storage (.+)").
		WithArgs("ortuman", "exodus:ns").
		WillReturnError(errMySQLStorage)

	elems, err = s.FetchPrivateXML("exodus:ns", "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
	require.Equal(t, 0, len(elems))
}

func TestMySQLStorageInsertOfflineMessages(t *testing.T) {
	j, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	message := xml.NewElementName("message")
	message.SetID(uuid.New())
	message.AppendElement(xml.NewElementName("body"))
	m, _ := xml.NewMessageFromElement(message, j, j)
	messageXML := m.String()

	s, mock := newMockMySQLStorage()
	mock.ExpectExec("INSERT INTO offline_messages (.+)").
		WithArgs("ortuman", messageXML).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := s.InsertOfflineMessage(m, "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectExec("INSERT INTO offline_messages (.+)").
		WithArgs("ortuman", messageXML).
		WillReturnError(errMySQLStorage)

	err = s.InsertOfflineMessage(m, "ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)
}

func TestMySQLStorageCountOfflineMessages(t *testing.T) {
	countColums := []string{"count"}

	s, mock := newMockMySQLStorage()
	mock.ExpectQuery("SELECT COUNT(.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(countColums).AddRow(1))

	cnt, _ := s.CountOfflineMessages("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, 1, cnt)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT COUNT(.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(countColums))

	cnt, _ = s.CountOfflineMessages("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, 0, cnt)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT COUNT(.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	_, err := s.CountOfflineMessages("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageFetchOfflineMessages(t *testing.T) {
	var offlineMessagesColumns = []string{"data"}

	s, mock := newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(offlineMessagesColumns).AddRow("<message id='abc'><body>Hi!</body></message>"))

	msgs, _ := s.FetchOfflineMessages("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, 1, len(msgs))

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(offlineMessagesColumns))

	msgs, _ = s.FetchOfflineMessages("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, 0, len(msgs))

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnRows(sqlmock.NewRows(offlineMessagesColumns).AddRow("<message id='abc'><body>Hi!"))

	_, err := s.FetchOfflineMessages("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.NotNil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectQuery("SELECT (.+) FROM offline_messages (.+)").
		WithArgs("ortuman").
		WillReturnError(errMySQLStorage)

	_, err = s.FetchOfflineMessages("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}

func TestMySQLStorageDeleteOfflineMessages(t *testing.T) {
	s, mock := newMockMySQLStorage()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.DeleteOfflineMessages("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Nil(t, err)

	s, mock = newMockMySQLStorage()
	mock.ExpectExec("DELETE FROM offline_messages (.+)").
		WithArgs("ortuman").WillReturnError(errMySQLStorage)

	err = s.DeleteOfflineMessages("ortuman")
	require.Nil(t, mock.ExpectationsWereMet())
	require.Equal(t, errMySQLStorage, err)
}
