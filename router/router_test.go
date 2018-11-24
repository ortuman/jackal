/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package router

import (
	"crypto/tls"
	"testing"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

type fakeS2SOut struct {
	elems []xmpp.XElement
}

func (f *fakeS2SOut) ID() string                     { return uuid.New() }
func (f *fakeS2SOut) SendElement(elem xmpp.XElement) { f.elems = append(f.elems, elem) }
func (f *fakeS2SOut) Disconnect(err error)           {}

type fakeS2SProvider struct {
	s2sOut *fakeS2SOut
}

func (f *fakeS2SProvider) GetS2SOut(localDomain, remoteDomain string) (stream.S2SOut, error) {
	return f.s2sOut, nil
}

func TestC2SManager(t *testing.T) {
	r, _, shutdown := setupTest()
	defer shutdown()

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

	r.Bind(strm1)
	r.Bind(strm2)
	r.Bind(strm3)
	r.Bind(strm4)
	r.Bind(strm5)

	require.Equal(t, 2, len(r.UserStreams("ortuman")))
	require.Equal(t, 1, len(r.UserStreams("hamlet")))
	require.Equal(t, 1, len(r.UserStreams("romeo")))
	require.Equal(t, 1, len(r.UserStreams("juliet")))

	r.Unbind(strm5)
	r.Unbind(strm4)
	r.Unbind(strm3)
	r.Unbind(strm2)
	r.Unbind(strm1)

	require.Equal(t, 0, len(r.UserStreams("ortuman")))
	require.Equal(t, 0, len(r.UserStreams("hamlet")))
	require.Equal(t, 0, len(r.UserStreams("romeo")))
	require.Equal(t, 0, len(r.UserStreams("juliet")))
}

func TestC2SManager_Routing(t *testing.T) {
	outS2S := fakeS2SOut{}
	s2sOutProvider := fakeS2SProvider{s2sOut: &outS2S}

	r, s, shutdown := setupTest()
	defer shutdown()

	r.SetS2SOutProvider(&s2sOutProvider)

	j1, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	j2, _ := jid.NewWithString("ortuman@jackal.im/garden", false)
	j3, _ := jid.NewWithString("hamlet@jackal.im/balcony", false)
	j4, _ := jid.NewWithString("hamlet@jackal.im/garden", false)
	j5, _ := jid.NewWithString("hamlet@jackal.im", false)
	j6, _ := jid.NewWithString("juliet@example.org/garden", false)
	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	stm3 := stream.NewMockC2S(uuid.New(), j3)

	r.Bind(stm1)
	r.Bind(stm2)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j6)

	// remote routing
	require.Nil(t, r.Route(iq))
	require.Equal(t, 1, len(outS2S.elems))

	iq.SetToJID(j3)
	require.Equal(t, ErrNotExistingAccount, r.Route(iq))

	s.EnableMockedError()
	require.Equal(t, memstorage.ErrMockedError, r.Route(iq))
	s.DisableMockedError()

	storage.InsertOrUpdateUser(&model.User{Username: "hamlet", Password: ""})
	require.Equal(t, ErrNotAuthenticated, r.Route(iq))

	stm4 := stream.NewMockC2S(uuid.New(), j4)
	r.Bind(stm4)
	require.Equal(t, ErrResourceNotFound, r.Route(iq))

	r.Bind(stm3)
	require.Nil(t, r.Route(iq))
	elem := stm3.FetchElement()
	require.Equal(t, iqID, elem.ID())

	// broadcast stanza
	iq.SetToJID(j5)
	require.Nil(t, r.Route(iq))
	elem = stm3.FetchElement()
	require.Equal(t, iqID, elem.ID())
	elem = stm4.FetchElement()
	require.Equal(t, iqID, elem.ID())

	// send clusterMessage to highest priority
	p1 := xmpp.NewElementName("presence")
	p1.SetFrom(j3.String())
	p1.SetTo(j3.String())
	p1.SetType(xmpp.AvailableType)
	pr1 := xmpp.NewElementName("priority")
	pr1.SetText("2")
	p1.AppendElement(pr1)
	presence1, _ := xmpp.NewPresenceFromElement(p1, j3, j3)
	stm3.SetPresence(presence1)

	p2 := xmpp.NewElementName("presence")
	p2.SetFrom(j4.String())
	p2.SetTo(j4.String())
	p2.SetType(xmpp.AvailableType)
	pr2 := xmpp.NewElementName("priority")
	pr2.SetText("1")
	p2.AppendElement(pr2)
	presence2, _ := xmpp.NewPresenceFromElement(p2, j4, j4)
	stm4.SetPresence(presence2)

	msgID := uuid.New()
	msg := xmpp.NewMessageType(msgID, xmpp.ChatType)
	msg.SetToJID(j5)
	require.Nil(t, r.Route(msg))
	elem = stm3.FetchElement()
	require.Equal(t, msgID, elem.ID())
}

func TestC2SManager_BlockedJID(t *testing.T) {
	r, _, shutdown := setupTest()
	defer shutdown()

	j1, _ := jid.NewWithString("ortuman@jackal.im/balcony", false)
	j2, _ := jid.NewWithString("hamlet@jackal.im/balcony", false)
	j3, _ := jid.NewWithString("hamlet@jackal.im/garden", false)
	j4, _ := jid.NewWithString("juliet@jackal.im/garden", false)
	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm2 := stream.NewMockC2S(uuid.New(), j2)

	r.Bind(stm1)
	r.Bind(stm2)

	// node + domain + resource
	bl1 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	}}
	storage.InsertBlockListItems(bl1)
	require.False(t, r.IsBlockedJID(j2, "ortuman"))
	require.True(t, r.IsBlockedJID(j3, "ortuman"))

	storage.DeleteBlockListItems(bl1)

	// node + domain
	bl2 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im",
	}}
	storage.InsertBlockListItems(bl2)
	r.ReloadBlockList("ortuman")

	require.True(t, r.IsBlockedJID(j2, "ortuman"))
	require.True(t, r.IsBlockedJID(j3, "ortuman"))
	require.False(t, r.IsBlockedJID(j4, "ortuman"))

	storage.DeleteBlockListItems(bl2)

	// domain + resource
	bl3 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "jackal.im/balcony",
	}}
	storage.InsertBlockListItems(bl3)
	r.ReloadBlockList("ortuman")

	require.True(t, r.IsBlockedJID(j2, "ortuman"))
	require.False(t, r.IsBlockedJID(j3, "ortuman"))
	require.False(t, r.IsBlockedJID(j4, "ortuman"))

	storage.DeleteBlockListItems(bl3)

	// domain
	bl4 := []model.BlockListItem{{
		Username: "ortuman",
		JID:      "jackal.im",
	}}
	storage.InsertBlockListItems(bl4)
	r.ReloadBlockList("ortuman")

	require.True(t, r.IsBlockedJID(j2, "ortuman"))
	require.True(t, r.IsBlockedJID(j3, "ortuman"))
	require.True(t, r.IsBlockedJID(j4, "ortuman"))

	storage.DeleteBlockListItems(bl4)

	// test blocked routing
	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j1)
	require.Equal(t, ErrBlockedJID, r.Route(iq))
}

func setupTest() (*Router, *memstorage.Storage, func()) {
	r, _ := New(&Config{
		Hosts: []HostConfig{{Name: "jackal.im", Certificate: tls.Certificate{}}},
	})
	s := memstorage.New()
	storage.Set(s)
	return r, s, func() {
		storage.Unset()
	}
}
