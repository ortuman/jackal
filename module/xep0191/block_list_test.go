/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0191

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/ortuman/jackal/router/host"

	c2srouter "github.com/ortuman/jackal/c2s/router"
	"github.com/ortuman/jackal/model"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module/xep0115"
	"github.com/ortuman/jackal/router"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0191_Matching(t *testing.T) {
	r, blockListRep, rosterRep := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(context.Background(), stm)

	ph := xep0115.New(r, nil)
	defer func() { _ = ph.Shutdown() }()

	x := New(nil, ph, r, rosterRep, blockListRep)
	defer func() { _ = x.Shutdown() }()

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
	r, blockListRep, rosterRep := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(context.Background(), stm)

	ph := xep0115.New(r, nil)
	defer func() { _ = ph.Shutdown() }()

	x := New(nil, ph, r, rosterRep, blockListRep)
	defer func() { _ = x.Shutdown() }()

	_ = blockListRep.InsertBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	})
	_ = blockListRep.InsertBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "jabber.org",
	})

	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j)
	iq1.AppendElement(xmpp.NewElementNamespace("blocklist", blockingCommandNamespace))

	x.ProcessIQ(context.Background(), iq1)
	elem := stm.ReceiveElement()
	bl := elem.Elements().ChildNamespace("blocklist", blockingCommandNamespace)
	require.NotNil(t, bl)
	require.Equal(t, 2, len(bl.Elements().Children("item")))

	requested, _ := stm.Value(xep191RequestedContextKey).(bool)
	require.True(t, requested)

	memorystorage.EnableMockedError()
	x.ProcessIQ(context.Background(), iq1)
	elem = stm.ReceiveElement()
	require.Len(t, elem.Error().Elements().All(), 1)
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	memorystorage.DisableMockedError()
}

func TestXEP191_BlockAndUnblock(t *testing.T) {
	r, blockListRep, rosterRep := setupTest("jackal.im")

	ph := xep0115.New(r, nil)
	defer func() { _ = ph.Shutdown() }()

	x := New(nil, ph, r, rosterRep, blockListRep)
	defer func() { _ = x.Shutdown() }()

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

	stm1.SetPresence(xmpp.NewPresence(j1, j1, xmpp.AvailableType))
	stm2.SetPresence(xmpp.NewPresence(j2, j2, xmpp.AvailableType))
	stm3.SetPresence(xmpp.NewPresence(j3, j3, xmpp.AvailableType))
	stm4.SetPresence(xmpp.NewPresence(j4, j4, xmpp.AvailableType))

	r.Bind(context.Background(), stm1)
	r.Bind(context.Background(), stm2)
	r.Bind(context.Background(), stm3)
	r.Bind(context.Background(), stm4)

	// register presences
	_, _ = ph.RegisterPresence(context.Background(), xmpp.NewPresence(j1, j1, xmpp.AvailableType))
	_, _ = ph.RegisterPresence(context.Background(), xmpp.NewPresence(j2, j2, xmpp.AvailableType))
	_, _ = ph.RegisterPresence(context.Background(), xmpp.NewPresence(j3, j3, xmpp.AvailableType))
	_, _ = ph.RegisterPresence(context.Background(), xmpp.NewPresence(j4, j4, xmpp.AvailableType))

	time.Sleep(time.Millisecond * 150) // wait until processed...

	stm1.SetValue(xep191RequestedContextKey, true)
	stm2.SetValue(xep191RequestedContextKey, true)

	_, _ = rosterRep.UpsertRosterItem(context.Background(), &rostermodel.Item{
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

	x.ProcessIQ(context.Background(), iq)
	elem := stm1.ReceiveElement()
	require.Len(t, elem.Error().Elements().All(), 1)
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	item := xmpp.NewElementName("item")
	item.SetAttribute("jid", "jackal.im/jail")
	block.AppendElement(item)
	iq.ClearElements()
	iq.AppendElement(block)

	// TEST BLOCK
	memorystorage.EnableMockedError()
	x.ProcessIQ(context.Background(), iq)
	elem = stm1.ReceiveElement()
	require.Len(t, elem.Error().Elements().All(), 1)
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	memorystorage.DisableMockedError()

	x.ProcessIQ(context.Background(), iq)

	// unavailable presence from *@jackal.im/jail
	elem = stm4.ReceiveElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xmpp.UnavailableType, elem.Type())
	require.Equal(t, "ortuman@jackal.im/balcony", elem.From())

	// result IQ
	elem = stm1.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// block IQ push
	elem = stm1.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.SetType, elem.Type())
	block2 := elem.Elements().ChildNamespace("block", blockingCommandNamespace)
	require.NotNil(t, block2)
	item2 := block.Elements().Child("item")
	require.NotNil(t, item2)

	elem = stm2.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.SetType, elem.Type())

	// check storage
	bl, _ := blockListRep.FetchBlockListItems(context.Background(), "ortuman")
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

	memorystorage.EnableMockedError()
	x.ProcessIQ(context.Background(), iq)
	elem = stm1.ReceiveElement()
	require.Len(t, elem.Error().Elements().All(), 1)
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	memorystorage.DisableMockedError()

	x.ProcessIQ(context.Background(), iq)

	// receive available presence from *@jackal.im/jail
	elem = stm4.ReceiveElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xmpp.AvailableType, elem.Type())
	require.Equal(t, "ortuman@jackal.im/balcony", elem.From())

	// result IQ
	elem = stm1.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// unblock IQ push
	elem = stm1.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.SetType, elem.Type())
	unblock2 := elem.Elements().ChildNamespace("unblock", blockingCommandNamespace)
	require.NotNil(t, block2)
	item2 = unblock2.Elements().Child("item")
	require.NotNil(t, item2)

	// test full unblock
	_ = blockListRep.InsertBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	})
	_ = blockListRep.InsertBlockListItem(context.Background(), &model.BlockListItem{
		Username: "ortuman",
		JID:      "jabber.org",
	})

	iqID = uuid.New()
	iq = xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1)
	unblock = xmpp.NewElementNamespace("unblock", blockingCommandNamespace)
	iq.AppendElement(unblock)

	x.ProcessIQ(context.Background(), iq)

	time.Sleep(time.Millisecond * 150) // wait until processed...

	blItems, _ := blockListRep.FetchBlockListItems(context.Background(), "ortuman")
	require.Equal(t, 0, len(blItems))
}

func setupTest(domain string) (router.Router, repository.BlockList, repository.Roster) {
	hosts, _ := host.New([]host.Config{{Name: domain, Certificate: tls.Certificate{}}})

	blockListRep := memorystorage.NewBlockList()
	rosterRep := memorystorage.NewRoster()
	r, _ := router.New(
		hosts,
		c2srouter.New(memorystorage.NewUser(), blockListRep),
		nil,
	)
	return r, blockListRep, rosterRep
}
