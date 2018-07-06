/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"testing"

	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

type fakeS2SOut struct {
	elems []xml.XElement
}

func (f *fakeS2SOut) ID() string                    { return uuid.New() }
func (f *fakeS2SOut) SendElement(elem xml.XElement) { f.elems = append(f.elems, elem) }
func (f *fakeS2SOut) Disconnect(err error)          {}

func TestC2SManager(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	Initialize(&Config{})
	defer func() {
		Shutdown()
		host.Shutdown()
	}()

	j1, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	j2, _ := jid.NewWithString("ortuman@jackal.im/garden", false)
	j3, _ := jid.NewWithString("hamlet@jackal.im/balcony", false)
	j4, _ := jid.NewWithString("romeo@jackal.im/balcony", false)
	j5, _ := jid.NewWithString("juliet@jackal.im/garden", false)
	strm1 := stream.NewMockC2S(uuid.New(), j1)
	strm2 := stream.NewMockC2S(uuid.New(), j2)
	strm3 := stream.NewMockC2S(uuid.New(), j3)
	strm4 := stream.NewMockC2S(uuid.New(), j4)
	strm5 := stream.NewMockC2S(uuid.New(), j5)

	Bind(strm1)
	Bind(strm2)
	Bind(strm3)
	Bind(strm4)
	Bind(strm5)

	require.Equal(t, 2, len(UserStreams("ortuman")))
	require.Equal(t, 1, len(UserStreams("hamlet")))
	require.Equal(t, 1, len(UserStreams("romeo")))
	require.Equal(t, 1, len(UserStreams("juliet")))

	Unbind(strm5)
	Unbind(strm4)
	Unbind(strm3)
	Unbind(strm2)
	Unbind(strm1)
}

func TestC2SManager_Routing(t *testing.T) {
	outS2S := fakeS2SOut{}
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	storage.Initialize(&storage.Config{Type: storage.Memory})
	Initialize(&Config{GetS2SOut: func(_, _ string) (stream.S2SOut, error) { return &outS2S, nil }})
	defer func() {
		Shutdown()
		storage.Shutdown()
		host.Shutdown()
	}()

	j1, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	j2, _ := jid.NewWithString("ortuman@jackal.im/garden", false)
	j3, _ := jid.NewWithString("hamlet@jackal.im/balcony", false)
	j4, _ := jid.NewWithString("hamlet@jackal.im/garden", false)
	j5, _ := jid.NewWithString("hamlet@jackal.im", false)
	j6, _ := jid.NewWithString("juliet@example.org/garden", false)
	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	stm3 := stream.NewMockC2S(uuid.New(), j3)

	Bind(stm1)
	Bind(stm2)

	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j6)

	// remote routing
	require.Nil(t, Route(iq))
	require.Equal(t, 1, len(outS2S.elems))

	iq.SetToJID(j3)
	require.Equal(t, ErrNotExistingAccount, Route(iq))

	storage.ActivateMockedError()
	require.Equal(t, memstorage.ErrMockedError, Route(iq))
	storage.DeactivateMockedError()

	storage.Instance().InsertOrUpdateUser(&model.User{Username: "hamlet", Password: ""})
	require.Equal(t, ErrNotAuthenticated, Route(iq))

	stm4 := stream.NewMockC2S(uuid.New(), j4)
	Bind(stm4)
	require.Equal(t, ErrResourceNotFound, Route(iq))

	Bind(stm3)
	require.Nil(t, Route(iq))
	elem := stm3.FetchElement()
	require.Equal(t, iqID, elem.ID())

	// broadcast stanza
	iq.SetToJID(j5)
	require.Nil(t, Route(iq))
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
	require.Nil(t, Route(msg))
	elem = stm3.FetchElement()
	require.Equal(t, msgID, elem.ID())
}

func TestC2SManager_BlockedJID(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	storage.Initialize(&storage.Config{Type: storage.Memory})
	Initialize(&Config{})
	defer func() {
		Shutdown()
		storage.Shutdown()
		host.Shutdown()
	}()

	j1, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	j2, _ := jid.NewWithString("hamlet@jackal.im/balcony", false)
	j3, _ := jid.NewWithString("hamlet@jackal.im/garden", false)
	j4, _ := jid.NewWithString("juliet@jackal.im/garden", false)
	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)

	Bind(stm1)
	Bind(stm2)

	// node + domain + resource
	bl1 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	}}
	storage.Instance().InsertBlockListItems(bl1)
	require.False(t, IsBlockedJID(j2, "ortuman"))
	require.True(t, IsBlockedJID(j3, "ortuman"))

	storage.Instance().DeleteBlockListItems(bl1)

	// node + domain
	bl2 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im",
	}}
	storage.Instance().InsertBlockListItems(bl2)
	ReloadBlockList("ortuman")

	require.True(t, IsBlockedJID(j2, "ortuman"))
	require.True(t, IsBlockedJID(j3, "ortuman"))
	require.False(t, IsBlockedJID(j4, "ortuman"))

	storage.Instance().DeleteBlockListItems(bl2)

	// domain + resource
	bl3 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "jackal.im/balcony",
	}}
	storage.Instance().InsertBlockListItems(bl3)
	ReloadBlockList("ortuman")

	require.True(t, IsBlockedJID(j2, "ortuman"))
	require.False(t, IsBlockedJID(j3, "ortuman"))
	require.False(t, IsBlockedJID(j4, "ortuman"))

	storage.Instance().DeleteBlockListItems(bl3)

	// domain
	bl4 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "jackal.im",
	}}
	storage.Instance().InsertBlockListItems(bl4)
	ReloadBlockList("ortuman")

	require.True(t, IsBlockedJID(j2, "ortuman"))
	require.True(t, IsBlockedJID(j3, "ortuman"))
	require.True(t, IsBlockedJID(j4, "ortuman"))

	storage.Instance().DeleteBlockListItems(bl4)

	// test blocked routing
	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j1)
	require.Equal(t, ErrBlockedJID, Route(iq))
}
