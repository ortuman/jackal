/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0054

import (
	"crypto/tls"
	"testing"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0054_Matching(t *testing.T) {
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	x := New(nil)
	defer x.Shutdown()

	// test MatchesIQ
	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j)

	vCard := xmpp.NewElementNamespace("query", vCardNamespace)

	iq.AppendElement(xmpp.NewElementNamespace("query", "jabber:client"))
	require.False(t, x.MatchesIQ(iq))
	iq.ClearElements()
	iq.AppendElement(vCard)
	require.False(t, x.MatchesIQ(iq))
	iq.SetToJID(j.ToBareJID())
	require.False(t, x.MatchesIQ(iq))
	vCard.SetName("vCard")
	require.True(t, x.MatchesIQ(iq))
}

func TestXEP0054_Set(t *testing.T) {
	_, _, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd", j)
	defer stm.Disconnect(nil)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())
	iq.AppendElement(testVCard())

	x := New(nil)
	defer x.Shutdown()

	x.ProcessIQ(iq, stm)
	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, xmpp.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	// set empty vCard...
	iq2ID := uuid.New()
	iq2 := xmpp.NewIQType(iq2ID, xmpp.SetType)
	iq2.SetFromJID(j)
	iq2.SetToJID(j.ToBareJID())
	iq2.AppendElement(xmpp.NewElementNamespace("vCard", vCardNamespace))

	x.ProcessIQ(iq2, stm)
	elem = stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, xmpp.ResultType, elem.Type())
	require.Equal(t, iq2ID, elem.ID())
}

func TestXEP0054_SetError(t *testing.T) {
	_, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("romeo", "jackal.im", "garden", true)

	stm := stream.NewMockC2S("abcd", j)
	defer stm.Disconnect(nil)

	x := New(nil)
	defer x.Shutdown()

	// set other user vCard...
	iq := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(j2.ToBareJID())
	iq.AppendElement(testVCard())

	x.ProcessIQ(iq, stm)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	// storage error
	s.EnableMockedError()
	defer s.DisableMockedError()

	iq2 := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iq2.SetFromJID(j)
	iq2.SetToJID(j.ToBareJID())
	iq2.AppendElement(testVCard())

	x.ProcessIQ(iq2, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0054_Get(t *testing.T) {
	_, _, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("romeo", "jackal.im", "garden", true)

	stm := stream.NewMockC2S("abcd", j)
	defer stm.Disconnect(nil)

	iqSet := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iqSet.SetFromJID(j)
	iqSet.SetToJID(j.ToBareJID())
	iqSet.AppendElement(testVCard())

	x := New(nil)
	defer x.Shutdown()

	x.ProcessIQ(iqSet, stm)
	_ = stm.ReceiveElement() // wait until set...

	iqGetID := uuid.New()
	iqGet := xmpp.NewIQType(iqGetID, xmpp.GetType)
	iqGet.SetFromJID(j)
	iqGet.SetToJID(j.ToBareJID())
	iqGet.AppendElement(xmpp.NewElementNamespace("vCard", vCardNamespace))

	x.ProcessIQ(iqGet, stm)
	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	vCard := elem.Elements().ChildNamespace("vCard", vCardNamespace)
	fn := vCard.Elements().Child("FN")
	require.Equal(t, "Forrest Gump", fn.Text())

	// non existing vCard...
	iqGet2ID := uuid.New()
	iqGet2 := xmpp.NewIQType(iqGet2ID, xmpp.GetType)
	iqGet2.SetFromJID(j2)
	iqGet2.SetToJID(j2.ToBareJID())
	iqGet2.AppendElement(xmpp.NewElementNamespace("vCard", vCardNamespace))

	x.ProcessIQ(iqGet2, stm)
	elem = stm.ReceiveElement()
	require.NotNil(t, elem)
	vCard = elem.Elements().ChildNamespace("vCard", vCardNamespace)
	require.Equal(t, 0, vCard.Elements().Count())
}

func TestXEP0054_GetError(t *testing.T) {
	_, s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd", j)
	defer stm.Disconnect(nil)

	iqSet := xmpp.NewIQType(uuid.New(), xmpp.SetType)
	iqSet.SetFromJID(j)
	iqSet.SetToJID(j.ToBareJID())
	iqSet.AppendElement(testVCard())

	x := New(nil)
	defer x.Shutdown()

	x.ProcessIQ(iqSet, stm)
	_ = stm.ReceiveElement() // wait until set...

	iqGetID := uuid.New()
	iqGet := xmpp.NewIQType(iqGetID, xmpp.GetType)
	iqGet.SetFromJID(j)
	iqGet.SetToJID(j.ToBareJID())
	vCard := xmpp.NewElementNamespace("vCard", vCardNamespace)
	vCard.AppendElement(xmpp.NewElementName("FN"))
	iqGet.AppendElement(vCard)

	x.ProcessIQ(iqGet, stm)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iqGet2ID := uuid.New()
	iqGet2 := xmpp.NewIQType(iqGet2ID, xmpp.GetType)
	iqGet2.SetFromJID(j)
	iqGet2.SetToJID(j.ToBareJID())
	iqGet2.AppendElement(xmpp.NewElementNamespace("vCard", vCardNamespace))

	s.EnableMockedError()
	defer s.DisableMockedError()

	x.ProcessIQ(iqGet2, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
}

func testVCard() xmpp.XElement {
	vCard := xmpp.NewElementNamespace("vCard", vCardNamespace)
	fn := xmpp.NewElementName("FN")
	fn.SetText("Forrest Gump")
	org := xmpp.NewElementName("ORG")
	org.SetText("Bubba Gump Shrimp Co.")
	vCard.AppendElement(fn)
	vCard.AppendElement(org)
	return vCard
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
