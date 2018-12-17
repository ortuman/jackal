/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0191

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/module/roster"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0191_Matching(t *testing.T) {
	rtr, _, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	r, rosterShutdownCh := roster.New(&roster.Config{}, rtr)
	defer close(rosterShutdownCh)

	x, shutdownCh := New(nil, r, rtr)
	defer close(shutdownCh)

	// test MatchesIQ
	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j)
	iq1.AppendElement(xmpp.NewElementNamespace("blocklist", blockingCommandNamespace))
	require.True(t, x.MatchesIQ(iq1))

	iq2 := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iq2.SetFromJID(j)
	iq2.SetToJID(j)
	iq2.AppendElement(xmpp.NewElementNamespace("block", blockingCommandNamespace))
	require.True(t, x.MatchesIQ(iq2))

	iq3 := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iq3.SetFromJID(j)
	iq3.SetToJID(j)
	iq3.AppendElement(xmpp.NewElementNamespace("unblock", blockingCommandNamespace))
	require.True(t, x.MatchesIQ(iq2))
}

func TestXEP0191_GetBlockList(t *testing.T) {
	rtr, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	defer stm.Disconnect(nil)

	r, rosterShutdownCh := roster.New(&roster.Config{}, rtr)
	defer close(rosterShutdownCh)

	x, shutdownCh := New(nil, r, rtr)
	defer close(shutdownCh)

	storage.InsertBlockListItems([]model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	}, {
		Username: "ortuman",
		JID:      "jabber.org",
	}})

	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j)
	iq1.AppendElement(xmpp.NewElementNamespace("blocklist", blockingCommandNamespace))

	x.ProcessIQ(iq1, stm)
	elem := stm.FetchElement()
	bl := elem.Elements().ChildNamespace("blocklist", blockingCommandNamespace)
	require.NotNil(t, bl)
	require.Equal(t, 2, len(bl.Elements().Children("item")))

	require.True(t, stm.GetBool(xep191RequestedContextKey))

	s.EnableMockedError()
	x.ProcessIQ(iq1, stm)
	elem = stm.FetchElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	s.DisableMockedError()
}

func TestXEP191_BlockAndUnblock(t *testing.T) {
	rtr, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	r, rosterShutdownCh := roster.New(&roster.Config{}, rtr)
	defer close(rosterShutdownCh)

	x, shutdownCh := New(nil, r, rtr)
	defer close(shutdownCh)

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	stm1 := stream.NewMockC2S(uuid.New(), j1)

	j2, _ := jid.New("ortuman", "jackal.im", "yard", true)
	stm2 := stream.NewMockC2S(uuid.New(), j2)

	j3, _ := jid.New("romeo", "jackal.im", "garden", true)
	stm3 := stream.NewMockC2S(uuid.New(), j3)

	j4, _ := jid.New("romeo", "jackal.im", "jail", true)
	stm4 := stream.NewMockC2S(uuid.New(), j4)

	stm1.SetAuthenticated(true)
	stm2.SetAuthenticated(true)
	stm3.SetAuthenticated(true)
	stm4.SetAuthenticated(true)

	rtr.Bind(stm1)
	rtr.Bind(stm2)
	rtr.Bind(stm3)
	rtr.Bind(stm4)

	// register presences
	r.ProcessPresence(xmpp.NewPresence(j1, j1, xmpp.AvailableType))
	r.ProcessPresence(xmpp.NewPresence(j2, j2, xmpp.AvailableType))
	r.ProcessPresence(xmpp.NewPresence(j3, j3, xmpp.AvailableType))
	r.ProcessPresence(xmpp.NewPresence(j4, j4, xmpp.AvailableType))

	time.Sleep(time.Millisecond * 150) // wait until processed...

	stm1.SetBool(xep191RequestedContextKey, true)
	stm2.SetBool(xep191RequestedContextKey, true)

	storage.InsertOrUpdateRosterItem(&rostermodel.Item{
		Username:     "ortuman",
		JID:          "romeo@jackal.im",
		Subscription: "both",
	})

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1)
	block := xmpp.NewElementNamespace("block", blockingCommandNamespace)
	iq.AppendElement(block)

	x.ProcessIQ(iq, stm1)
	elem := stm1.FetchElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	item := xmpp.NewElementName("item")
	item.SetAttribute("jid", "jackal.im/jail")
	block.AppendElement(item)
	iq.ClearElements()
	iq.AppendElement(block)

	// TEST BLOCK
	s.EnableMockedError()
	x.ProcessIQ(iq, stm1)
	elem = stm1.FetchElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	s.DisableMockedError()

	x.ProcessIQ(iq, stm1)

	// unavailable presence from *@jackal.im/jail
	elem = stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xmpp.UnavailableType, elem.Type())
	require.Equal(t, "romeo@jackal.im/jail", elem.From())

	// result IQ
	elem = stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// block IQ push
	elem = stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.SetType, elem.Type())
	block2 := elem.Elements().ChildNamespace("block", blockingCommandNamespace)
	require.NotNil(t, block2)
	item2 := block.Elements().Child("item")
	require.NotNil(t, item2)

	// ortuman@jackal.im/yard
	elem = stm2.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xmpp.UnavailableType, elem.Type())
	require.Equal(t, "romeo@jackal.im/jail", elem.From())

	elem = stm2.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.SetType, elem.Type())

	// check storage
	bl, _ := storage.FetchBlockListItems("ortuman")
	require.NotNil(t, bl)
	require.Equal(t, 1, len(bl))
	require.Equal(t, "jackal.im/jail", bl[0].JID)

	// TEST UNBLOCK
	iqID = uuid.New()
	iq = xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1)
	unblock := xmpp.NewElementNamespace("unblock", blockingCommandNamespace)
	item = xmpp.NewElementName("item")
	item.SetAttribute("jid", "jackal.im/jail")
	unblock.AppendElement(item)
	iq.AppendElement(unblock)

	s.EnableMockedError()
	x.ProcessIQ(iq, stm1)
	elem = stm1.FetchElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	s.DisableMockedError()

	x.ProcessIQ(iq, stm1)

	// receive available presence from *@jackal.im/jail
	elem = stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xmpp.AvailableType, elem.Type())
	require.Equal(t, "romeo@jackal.im/jail", elem.From())

	// result IQ
	elem = stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// unblock IQ push
	elem = stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.SetType, elem.Type())
	unblock2 := elem.Elements().ChildNamespace("unblock", blockingCommandNamespace)
	require.NotNil(t, block2)
	item2 = unblock2.Elements().Child("item")
	require.NotNil(t, item2)

	// test full unblock
	storage.InsertBlockListItems([]model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	}, {
		Username: "ortuman",
		JID:      "jabber.org",
	}})

	iqID = uuid.New()
	iq = xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1)
	unblock = xmpp.NewElementNamespace("unblock", blockingCommandNamespace)
	iq.AppendElement(unblock)

	x.ProcessIQ(iq, stm1)

	time.Sleep(time.Millisecond * 150) // wait until processed...

	blItms, _ := storage.FetchBlockListItems("ortuman")
	require.Equal(t, 0, len(blItms))
}

func setupTest(domain string) (*router.Router, *memstorage.Storage, func()) {
	r, _ := router.New(&router.Config{
		Hosts: []router.HostConfig{{Name: domain, Certificate: tls.Certificate{}}},
	})
	s := memstorage.New()
	storage.Set(s)
	return r, s, func() {
		storage.Unset()
	}
}
