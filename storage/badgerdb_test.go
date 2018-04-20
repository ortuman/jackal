/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package storage

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

type testBadgerDBHelper struct {
	db      *badgerDB
	dataDir string
}

func TestBadgerDB_User(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	usr := model.User{Username: "ortuman", Password: "1234"}

	err := h.db.InsertOrUpdateUser(&usr)
	require.Nil(t, err)

	usr2, err := h.db.FetchUser("ortuman")
	require.Nil(t, err)
	require.Equal(t, "ortuman", usr2.Username)
	require.Equal(t, "1234", usr2.Password)

	exists, err := h.db.UserExists("ortuman")
	require.Nil(t, err)
	require.True(t, exists)

	usr3, err := h.db.FetchUser("ortuman2")
	require.Nil(t, usr3)
	require.Nil(t, err)

	err = h.db.DeleteUser("ortuman")
	require.Nil(t, err)

	exists, err = h.db.UserExists("ortuman")
	require.Nil(t, err)
	require.False(t, exists)
}

func TestBadgerDB_VCard(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	vcard := xml.NewElementNamespace("vCard", "vcard-temp")
	fn := xml.NewElementName("FN")
	fn.SetText("Miguel Ángel Ortuño")
	vcard.AppendElement(fn)

	err := h.db.InsertOrUpdateVCard(vcard, "ortuman")
	require.Nil(t, err)

	vcard2, err := h.db.FetchVCard("ortuman")
	require.Nil(t, err)
	require.Equal(t, "vCard", vcard2.Name())
	require.Equal(t, "vcard-temp", vcard2.Namespace())
	require.NotNil(t, vcard2.Elements().Child("FN"))

	vcard3, err := h.db.FetchVCard("ortuman2")
	require.Nil(t, vcard3)
	require.Nil(t, err)
}

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

func TestBadgerDB_RosterItems(t *testing.T) {
	t.Parallel()

	h := tUtilBadgerDBSetup()
	defer tUtilBadgerDBTeardown(h)

	ri1 := &model.RosterItem{
		User:         "ortuman",
		Contact:      "juliet",
		Subscription: "both",
	}
	ri2 := &model.RosterItem{
		User:         "ortuman",
		Contact:      "romeo",
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

	rn1 := model.RosterNotification{
		User:     "juliet",
		Contact:  "ortuman",
		Elements: []xml.XElement{},
	}
	rn2 := model.RosterNotification{
		User:     "romeo",
		Contact:  "ortuman",
		Elements: []xml.XElement{},
	}
	require.NoError(t, h.db.InsertOrUpdateRosterNotification(&rn1))
	require.NoError(t, h.db.InsertOrUpdateRosterNotification(&rn2))

	rns, err := h.db.FetchRosterNotifications("ortuman")
	require.Nil(t, err)
	require.Equal(t, 2, len(rns))

	rns2, err := h.db.FetchRosterNotifications("ortuman2")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns2))

	require.NoError(t, h.db.DeleteRosterNotification(rn1.User, rn1.Contact))

	rns, err = h.db.FetchRosterNotifications("ortuman")
	require.Nil(t, err)
	require.Equal(t, 1, len(rns))

	require.NoError(t, h.db.DeleteRosterNotification(rn2.User, rn2.Contact))

	rns, err = h.db.FetchRosterNotifications("ortuman")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))
}

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

func tUtilBadgerDBSetup() *testBadgerDBHelper {
	h := &testBadgerDBHelper{}
	dir, _ := ioutil.TempDir("", "")
	h.dataDir = dir + "/com.jackal.tests.badgerdb." + uuid.New()
	cfg := config.BadgerDb{DataDir: h.dataDir}
	h.db = newBadgerDB(&cfg)
	return h
}

func tUtilBadgerDBTeardown(h *testBadgerDBHelper) {
	h.db.Shutdown()
	os.RemoveAll(h.dataDir)
}
