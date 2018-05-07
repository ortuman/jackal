/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"testing"

	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestC2SManager(t *testing.T) {
	Initialize(&Config{Domains: []string{"jackal.im"}})
	defer Shutdown()

	require.Equal(t, "jackal.im", Instance().DefaultLocalDomain())
	require.True(t, Instance().IsLocalDomain("jackal.im"))
	require.False(t, Instance().IsLocalDomain("example.org"))

	j1, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	j2, _ := xml.NewJIDString("ortuman@jackal.im/garden", false)
	j3, _ := xml.NewJIDString("hamlet@jackal.im/balcony", false)
	j4, _ := xml.NewJIDString("romeo@jackal.im/balcony", false)
	j5, _ := xml.NewJIDString("juliet@jackal.im/garden", false)
	j6, _ := xml.NewJIDString("juliet@example.org/garden", false)
	strm1 := NewMockStream(uuid.New(), j1)
	strm2 := NewMockStream(uuid.New(), j2)
	strm3 := NewMockStream(uuid.New(), j3)
	strm4 := NewMockStream(uuid.New(), j4)
	strm5 := NewMockStream(uuid.New(), j5)
	strm6 := NewMockStream(uuid.New(), j6)

	err := Instance().RegisterStream(strm1)
	require.Nil(t, err)
	err = Instance().RegisterStream(strm1) // already registered...
	require.NotNil(t, err)
	err = Instance().RegisterStream(strm2)
	require.Nil(t, err)
	err = Instance().RegisterStream(strm3)
	require.Nil(t, err)
	err = Instance().RegisterStream(strm4)
	require.Nil(t, err)
	err = Instance().RegisterStream(strm5)
	require.Nil(t, err)
	err = Instance().RegisterStream(strm6)
	require.NotNil(t, err)

	strm1.SetResource("")
	err = Instance().AuthenticateStream(strm1) // resource not assigned...
	require.NotNil(t, err)
	strm1.SetResource("balcony")
	err = Instance().AuthenticateStream(strm1)
	require.Nil(t, err)
	err = Instance().AuthenticateStream(strm2)
	require.Nil(t, err)
	err = Instance().AuthenticateStream(strm3)
	require.Nil(t, err)
	err = Instance().AuthenticateStream(strm4)
	require.Nil(t, err)
	err = Instance().AuthenticateStream(strm5)
	require.Nil(t, err)

	strms := Instance().StreamsMatchingJID(j1.ToBareJID())
	require.Equal(t, 2, len(strms))
	require.Equal(t, "ortuman@jackal.im/balcony", strms[0].JID().String())
	require.Equal(t, "ortuman@jackal.im/garden", strms[1].JID().String())

	mj1, _ := xml.NewJIDString("jackal.im", true)
	strms = Instance().StreamsMatchingJID(mj1)
	require.Equal(t, 5, len(strms))

	mj2, _ := xml.NewJIDString("jackal.im/balcony", true)
	strms = Instance().StreamsMatchingJID(mj2)
	require.Equal(t, 3, len(strms))

	mj3, _ := xml.NewJIDString("example.org", true)
	strms = Instance().StreamsMatchingJID(mj3)
	require.Nil(t, strms)

	err = Instance().UnregisterStream(strm1)
	require.Nil(t, err)
	err = Instance().UnregisterStream(strm1)
	require.NotNil(t, err) // already unregistered...
	err = Instance().UnregisterStream(strm2)
	require.Nil(t, err)

	strms = Instance().StreamsMatchingJID(j1.ToBareJID())
	require.Equal(t, 0, len(strms))
}

func TestC2SManager_Routing(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Mock})
	defer storage.Shutdown()

	Initialize(&Config{Domains: []string{"jackal.im"}})
	defer Shutdown()

	j1, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	j2, _ := xml.NewJIDString("ortuman@jackal.im/garden", false)
	j3, _ := xml.NewJIDString("hamlet@jackal.im/balcony", false)
	j4, _ := xml.NewJIDString("hamlet@jackal.im/garden", false)
	j5, _ := xml.NewJIDString("hamlet@jackal.im", false)
	j6, _ := xml.NewJIDString("juliet@example.org/garden", false)
	stm1 := NewMockStream(uuid.New(), j1)
	stm2 := NewMockStream(uuid.New(), j2)
	stm3 := NewMockStream(uuid.New(), j3)

	Instance().RegisterStream(stm1)
	Instance().RegisterStream(stm2)
	Instance().AuthenticateStream(stm1)
	Instance().AuthenticateStream(stm2)

	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j6)
	require.Nil(t, Instance().Route(iq))

	iq.SetToJID(j3)
	require.Equal(t, ErrNotExistingAccount, Instance().Route(iq))

	storage.ActivateMockedError()
	require.Equal(t, storage.ErrMockedError, Instance().Route(iq))
	storage.DeactivateMockedError()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "hamlet", Password: ""})
	require.Equal(t, ErrNotAuthenticated, Instance().Route(iq))

	stm4 := NewMockStream(uuid.New(), j4)
	Instance().RegisterStream(stm4)
	Instance().AuthenticateStream(stm4)
	require.Equal(t, ErrResourceNotFound, Instance().Route(iq))

	Instance().RegisterStream(stm3)
	Instance().AuthenticateStream(stm3)
	require.Nil(t, Instance().Route(iq))
	elem := stm3.FetchElement()
	require.Equal(t, iqID, elem.ID())

	// broadcast stanza
	iq.SetToJID(j5)
	require.Nil(t, Instance().Route(iq))
	elem = stm3.FetchElement()
	require.Equal(t, iqID, elem.ID())
	elem = stm4.FetchElement()
	require.Equal(t, iqID, elem.ID())

	// send message to highest priority
	p1 := xml.NewElementName("presence")
	p1.SetFrom(j3.String())
	p1.SetTo(j3.String())
	p1.SetType(xml.AvailableType)
	pr1 := xml.NewElementName("priority")
	pr1.SetText("2")
	p1.AppendElement(pr1)
	presence1, _ := xml.NewPresenceFromElement(p1, j3, j3)
	stm3.SetPresence(presence1)

	p2 := xml.NewElementName("presence")
	p2.SetFrom(j4.String())
	p2.SetTo(j4.String())
	p2.SetType(xml.AvailableType)
	pr2 := xml.NewElementName("priority")
	pr2.SetText("1")
	p2.AppendElement(pr2)
	presence2, _ := xml.NewPresenceFromElement(p2, j4, j4)
	stm4.SetPresence(presence2)

	msgID := uuid.New()
	msg := xml.NewMessageType(msgID, xml.ChatType)
	msg.SetToJID(j5)
	require.Nil(t, Instance().Route(msg))
	elem = stm3.FetchElement()
	require.Equal(t, msgID, elem.ID())
}

func TestC2SManager_StreamsMatching(t *testing.T) {
	Initialize(&Config{Domains: []string{"jackal.im"}})
	defer Shutdown()

	j1, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	j2, _ := xml.NewJIDString("ortuman@jackal.im/garden", false)
	j3, _ := xml.NewJIDString("hamlet@jackal.im/garden", false)
	j4, _ := xml.NewJIDString("juliet@jackal.im/garden", false)
	stm1 := NewMockStream(uuid.New(), j1)
	stm2 := NewMockStream(uuid.New(), j2)
	stm3 := NewMockStream(uuid.New(), j3)
	stm4 := NewMockStream(uuid.New(), j4)

	Instance().RegisterStream(stm1)
	Instance().RegisterStream(stm2)
	Instance().RegisterStream(stm3)
	Instance().RegisterStream(stm4)
	Instance().AuthenticateStream(stm1)
	Instance().AuthenticateStream(stm2)
	Instance().AuthenticateStream(stm3)
	Instance().AuthenticateStream(stm4)

	j, _ := xml.NewJIDString("ortuman@jackal.im/garden", true)
	require.Equal(t, 1, len(Instance().StreamsMatchingJID(j)))

	j, _ = xml.NewJIDString("ortuman@jackal.im", true)
	require.Equal(t, 2, len(Instance().StreamsMatchingJID(j)))

	j, _ = xml.NewJIDString("jackal.im/garden", true)
	require.Equal(t, 3, len(Instance().StreamsMatchingJID(j)))
}

func TestC2SManager_BlockedJID(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Mock})
	defer storage.Shutdown()

	Initialize(&Config{Domains: []string{"jackal.im"}})
	defer Shutdown()

	j1, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	j2, _ := xml.NewJIDString("hamlet@jackal.im/balcony", false)
	j3, _ := xml.NewJIDString("hamlet@jackal.im/garden", false)
	j4, _ := xml.NewJIDString("juliet@jackal.im/garden", false)
	stm1 := NewMockStream(uuid.New(), j1)
	stm2 := NewMockStream(uuid.New(), j2)

	Instance().RegisterStream(stm1)
	Instance().RegisterStream(stm2)
	Instance().AuthenticateStream(stm1)
	Instance().AuthenticateStream(stm2)

	// node + domain + resource
	bl1 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	}}
	storage.Instance().InsertOrUpdateBlockListItems(bl1)
	require.False(t, Instance().IsBlockedJID(j2, "ortuman"))
	require.True(t, Instance().IsBlockedJID(j3, "ortuman"))

	storage.Instance().DeleteBlockListItems(bl1)

	// node + domain
	bl2 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im",
	}}
	storage.Instance().InsertOrUpdateBlockListItems(bl2)
	Instance().ReloadBlockList("ortuman")

	require.True(t, Instance().IsBlockedJID(j2, "ortuman"))
	require.True(t, Instance().IsBlockedJID(j3, "ortuman"))
	require.False(t, Instance().IsBlockedJID(j4, "ortuman"))

	storage.Instance().DeleteBlockListItems(bl2)

	// domain + resource
	bl3 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "jackal.im/balcony",
	}}
	storage.Instance().InsertOrUpdateBlockListItems(bl3)
	Instance().ReloadBlockList("ortuman")

	require.True(t, Instance().IsBlockedJID(j2, "ortuman"))
	require.False(t, Instance().IsBlockedJID(j3, "ortuman"))
	require.False(t, Instance().IsBlockedJID(j4, "ortuman"))

	storage.Instance().DeleteBlockListItems(bl3)

	// domain
	bl4 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "jackal.im",
	}}
	storage.Instance().InsertOrUpdateBlockListItems(bl4)
	Instance().ReloadBlockList("ortuman")

	require.True(t, Instance().IsBlockedJID(j2, "ortuman"))
	require.True(t, Instance().IsBlockedJID(j3, "ortuman"))
	require.True(t, Instance().IsBlockedJID(j4, "ortuman"))

	storage.Instance().DeleteBlockListItems(bl4)

	// test blocked routing
	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j1)
	require.Equal(t, ErrBlockedJID, Instance().Route(iq))
}
