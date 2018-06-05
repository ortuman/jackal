/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0191

import (
	"testing"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0191_Matching(t *testing.T) {
	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	x := New(nil)

	// test MatchesIQ
	iq1 := xml.NewIQType(uuid.New(), xml.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j)
	iq1.AppendElement(xml.NewElementNamespace("blocklist", blockingCommandNamespace))
	require.True(t, x.MatchesIQ(iq1))

	iq2 := xml.NewIQType(uuid.New(), xml.SetType)
	iq2.SetFromJID(j)
	iq2.SetToJID(j)
	iq2.AppendElement(xml.NewElementNamespace("block", blockingCommandNamespace))
	require.True(t, x.MatchesIQ(iq2))

	iq3 := xml.NewIQType(uuid.New(), xml.SetType)
	iq3.SetFromJID(j)
	iq3.SetToJID(j)
	iq3.AppendElement(xml.NewElementNamespace("unblock", blockingCommandNamespace))
	require.True(t, x.MatchesIQ(iq2))
}

func TestXEP0191_GetBlockList(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	defer stm.Disconnect(nil)

	x := New(stm)

	storage.Instance().InsertBlockListItems([]model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	}, {
		Username: "ortuman",
		JID:      "jabber.org",
	}})

	iq1 := xml.NewIQType(uuid.New(), xml.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j)
	iq1.AppendElement(xml.NewElementNamespace("blocklist", blockingCommandNamespace))

	x.ProcessIQ(iq1)
	elem := stm.FetchElement()
	bl := elem.Elements().ChildNamespace("blocklist", blockingCommandNamespace)
	require.NotNil(t, bl)
	require.Equal(t, 2, len(bl.Elements().Children("item")))

	require.True(t, stm.Context().Bool(xep191RequestedContextKey))

	storage.ActivateMockedError()
	x.ProcessIQ(iq1)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	storage.DeactivateMockedError()
}

func TestXEP191_BlockAndUnblock(t *testing.T) {
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	stm1 := stream.NewMockC2S(uuid.New(), j1)

	x := New(stm1)

	j2, _ := xml.NewJID("ortuman", "jackal.im", "yard", true)
	stm2 := stream.NewMockC2S(uuid.New(), j2)

	j3, _ := xml.NewJID("romeo", "jackal.im", "garden", true)
	stm3 := stream.NewMockC2S(uuid.New(), j3)

	j4, _ := xml.NewJID("romeo", "jackal.im", "jail", true)
	stm4 := stream.NewMockC2S(uuid.New(), j4)

	router.Instance().RegisterC2S(stm1)
	router.Instance().RegisterC2S(stm2)
	router.Instance().RegisterC2S(stm3)
	router.Instance().RegisterC2S(stm4)
	router.Instance().RegisterC2SResource(stm1)
	router.Instance().RegisterC2SResource(stm2)
	router.Instance().RegisterC2SResource(stm3)
	router.Instance().RegisterC2SResource(stm4)

	stm1.SetAuthenticated(true)
	stm2.SetAuthenticated(true)
	stm3.SetAuthenticated(true)
	stm4.SetAuthenticated(true)

	stm1.Context().SetBool(true, xep191RequestedContextKey)
	stm2.Context().SetBool(true, xep191RequestedContextKey)

	storage.Instance().InsertOrUpdateRosterItem(&model.RosterItem{
		Username:     "ortuman",
		JID:          "romeo@jackal.im",
		Subscription: "both",
	})

	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1)
	block := xml.NewElementNamespace("block", blockingCommandNamespace)
	iq.AppendElement(block)

	x.ProcessIQ(iq)
	elem := stm1.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	item := xml.NewElementName("item")
	item.SetAttribute("jid", "jackal.im/jail")
	block.AppendElement(item)
	iq.ClearElements()
	iq.AppendElement(block)

	// TEST BLOCK
	storage.ActivateMockedError()
	x.ProcessIQ(iq)
	elem = stm1.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	storage.DeactivateMockedError()

	x.ProcessIQ(iq)

	// unavailable presence from *@jackal.im/jail
	elem = stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.UnavailableType, elem.Type())
	require.Equal(t, "romeo@jackal.im/jail", elem.From())

	// result IQ
	elem = stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xml.ResultType, elem.Type())

	// block IQ push
	elem = stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	block2 := elem.Elements().ChildNamespace("block", blockingCommandNamespace)
	require.NotNil(t, block2)
	item2 := block.Elements().Child("item")
	require.NotNil(t, item2)

	// ortuman@jackal.im/yard
	elem = stm2.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.UnavailableType, elem.Type())
	require.Equal(t, "romeo@jackal.im/jail", elem.From())

	elem = stm2.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())

	// check storage
	bl, _ := storage.Instance().FetchBlockListItems("ortuman")
	require.NotNil(t, bl)
	require.Equal(t, 1, len(bl))
	require.Equal(t, "jackal.im/jail", bl[0].JID)

	// TEST UNBLOCK
	iqID = uuid.New()
	iq = xml.NewIQType(iqID, xml.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1)
	unblock := xml.NewElementNamespace("unblock", blockingCommandNamespace)
	item = xml.NewElementName("item")
	item.SetAttribute("jid", "jackal.im/jail")
	unblock.AppendElement(item)
	iq.AppendElement(unblock)

	storage.ActivateMockedError()
	x.ProcessIQ(iq)
	elem = stm1.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	storage.DeactivateMockedError()

	x.ProcessIQ(iq)

	// receive available presence from *@jackal.im/jail
	elem = stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.AvailableType, elem.Type())
	require.Equal(t, "romeo@jackal.im/jail", elem.From())

	// result IQ
	elem = stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, iqID, elem.ID())
	require.Equal(t, xml.ResultType, elem.Type())

	// unblock IQ push
	elem = stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	unblock2 := elem.Elements().ChildNamespace("unblock", blockingCommandNamespace)
	require.NotNil(t, block2)
	item2 = unblock2.Elements().Child("item")
	require.NotNil(t, item2)

	// test full unblock
	storage.Instance().InsertBlockListItems([]model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	}, {
		Username: "ortuman",
		JID:      "jabber.org",
	}})

	iqID = uuid.New()
	iq = xml.NewIQType(iqID, xml.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1)
	unblock = xml.NewElementNamespace("unblock", blockingCommandNamespace)
	iq.AppendElement(unblock)

	x.ProcessIQ(iq)

	blItms, _ := storage.Instance().FetchBlockListItems("ortuman")
	require.Equal(t, 0, len(blItms))
}
