/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0054

import (
	"testing"

	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0054_Matching(t *testing.T) {
	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	x := New(nil)

	require.Equal(t, []string{vCardNamespace}, x.AssociatedNamespaces())

	// test MatchesIQ
	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	iq.SetFromJID(j)

	vCard := xml.NewElementNamespace("query", vCardNamespace)

	iq.AppendElement(xml.NewElementNamespace("query", "jabber:client"))
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
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	stm := c2s.NewMockStream("abcd", j)

	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())
	iq.AppendElement(testVCard())

	x := New(stm)

	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.NotNil(t, elem)
	require.Equal(t, xml.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	// set empty vCard...
	iq2ID := uuid.New()
	iq2 := xml.NewIQType(iq2ID, xml.SetType)
	iq2.SetFromJID(j)
	iq2.SetToJID(j.ToBareJID())
	iq2.AppendElement(xml.NewElementNamespace("vCard", vCardNamespace))

	x.ProcessIQ(iq2)
	elem = stm.FetchElement()
	require.NotNil(t, elem)
	require.Equal(t, xml.ResultType, elem.Type())
	require.Equal(t, iq2ID, elem.ID())
}

func TestXEP0054_SetError(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	j2, _ := xml.NewJID("romeo", "jackal.im", "garden", true)
	stm := c2s.NewMockStream("abcd", j)
	stm.SetUsername("ortuman")

	x := New(stm)

	// set other user vCard...
	iq := xml.NewIQType(uuid.New(), xml.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(j2.ToBareJID())
	iq.AppendElement(testVCard())

	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	// storage error
	storage.ActivateMockedError()
	defer storage.DeactivateMockedError()

	iq2 := xml.NewIQType(uuid.New(), xml.SetType)
	iq2.SetFromJID(j)
	iq2.SetToJID(j.ToBareJID())
	iq2.AppendElement(testVCard())

	x.ProcessIQ(iq2)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0054_Get(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	j2, _ := xml.NewJID("romeo", "jackal.im", "garden", true)
	stm := c2s.NewMockStream("abcd", j)

	iqSet := xml.NewIQType(uuid.New(), xml.SetType)
	iqSet.SetFromJID(j)
	iqSet.SetToJID(j.ToBareJID())
	iqSet.AppendElement(testVCard())

	x := New(stm)

	x.ProcessIQ(iqSet)
	_ = stm.FetchElement() // wait until set...

	iqGetID := uuid.New()
	iqGet := xml.NewIQType(iqGetID, xml.GetType)
	iqGet.SetFromJID(j)
	iqGet.SetToJID(j.ToBareJID())
	iqGet.AppendElement(xml.NewElementNamespace("vCard", vCardNamespace))

	x.ProcessIQ(iqGet)
	elem := stm.FetchElement()
	require.NotNil(t, elem)
	vCard := elem.Elements().ChildNamespace("vCard", vCardNamespace)
	fn := vCard.Elements().Child("FN")
	require.Equal(t, "Forrest Gump", fn.Text())

	// non existing vCard...
	iqGet2ID := uuid.New()
	iqGet2 := xml.NewIQType(iqGet2ID, xml.GetType)
	iqGet2.SetFromJID(j2)
	iqGet2.SetToJID(j2.ToBareJID())
	iqGet2.AppendElement(xml.NewElementNamespace("vCard", vCardNamespace))

	x.ProcessIQ(iqGet2)
	elem = stm.FetchElement()
	require.NotNil(t, elem)
	vCard = elem.Elements().ChildNamespace("vCard", vCardNamespace)
	require.Equal(t, 0, vCard.Elements().Count())
}

func TestXEP0054_GetError(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	stm := c2s.NewMockStream("abcd", j)

	iqSet := xml.NewIQType(uuid.New(), xml.SetType)
	iqSet.SetFromJID(j)
	iqSet.SetToJID(j.ToBareJID())
	iqSet.AppendElement(testVCard())

	x := New(stm)

	x.ProcessIQ(iqSet)
	_ = stm.FetchElement() // wait until set...

	iqGetID := uuid.New()
	iqGet := xml.NewIQType(iqGetID, xml.GetType)
	iqGet.SetFromJID(j)
	iqGet.SetToJID(j.ToBareJID())
	vCard := xml.NewElementNamespace("vCard", vCardNamespace)
	vCard.AppendElement(xml.NewElementName("FN"))
	iqGet.AppendElement(vCard)

	x.ProcessIQ(iqGet)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iqGet2ID := uuid.New()
	iqGet2 := xml.NewIQType(iqGet2ID, xml.GetType)
	iqGet2.SetFromJID(j)
	iqGet2.SetToJID(j.ToBareJID())
	iqGet2.AppendElement(xml.NewElementNamespace("vCard", vCardNamespace))

	storage.ActivateMockedError()
	defer storage.DeactivateMockedError()

	x.ProcessIQ(iqGet2)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
}

func testVCard() xml.XElement {
	vCard := xml.NewElementNamespace("vCard", vCardNamespace)
	fn := xml.NewElementName("FN")
	fn.SetText("Forrest Gump")
	org := xml.NewElementName("ORG")
	org.SetText("Bubba Gump Shrimp Co.")
	vCard.AppendElement(fn)
	vCard.AppendElement(org)
	return vCard
}
