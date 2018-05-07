/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"fmt"
	"testing"

	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0030_Matching(t *testing.T) {
	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	x := New(nil)
	defer x.Done()

	for _, ns := range x.AssociatedNamespaces() {
		switch ns {
		case discoInfoNamespace, discoItemsNamespace:
			continue
		default:
			require.Fail(t, fmt.Sprintf("unrecognized XEPDiscoInfo namespace: %s", ns))
			return
		}
	}
	// test MatchesIQ
	iq1 := xml.NewIQType(uuid.New(), xml.GetType)
	iq1.SetFromJID(j)

	require.False(t, x.MatchesIQ(iq1))

	iq1.AppendElement(xml.NewElementNamespace("query", discoItemsNamespace))

	iq2 := xml.NewIQType(uuid.New(), xml.GetType)
	iq2.SetFromJID(j)
	iq2.AppendElement(xml.NewElementNamespace("query", discoItemsNamespace))

	require.True(t, x.MatchesIQ(iq1))
	require.True(t, x.MatchesIQ(iq2))

	iq1.SetType(xml.SetType)
	iq2.SetType(xml.ResultType)

	require.False(t, x.MatchesIQ(iq1))
	require.False(t, x.MatchesIQ(iq2))
}

func TestXEP0030_SetItems(t *testing.T) {
	x := New(nil)
	defer x.Done()

	its := []DiscoItem{
		{Jid: "j1@jackal.im", Name: "a name", Node: "node1"},
		{Jid: "j2@jackal.im", Name: "a second name", Node: "node2"},
	}
	x.SetItems(its)
	require.Equal(t, its, x.Items())
}

func TestXEP0030_SetIdentities(t *testing.T) {
	x := New(nil)
	defer x.Done()

	ids := []DiscoIdentity{{
		Category: "server",
		Type:     "im",
		Name:     "default",
	}}
	x.SetIdentities(ids)
	require.Equal(t, ids, x.Identities())
}

func TestXEP0030_SetFeatures(t *testing.T) {
	x := New(nil)
	defer x.Done()

	fs := []DiscoFeature{
		discoInfoNamespace,
		discoItemsNamespace,
	}
	x.SetFeatures(fs)
	require.Equal(t, fs, x.Features())
}

func TestXEP0030_BadToJID(t *testing.T) {
	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	stm := c2s.NewMockStream("abcd", j)

	x := New(stm)
	defer x.Done()

	iq1 := xml.NewIQType(uuid.New(), xml.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j)
	iq1.AppendElement(xml.NewElementNamespace("query", discoItemsNamespace))

	x.ProcessIQ(iq1)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrFeatureNotImplemented.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0030_GetFeatures(t *testing.T) {
	srvJid, _ := xml.NewJID("", "jackal.im", "", true)

	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	stm := c2s.NewMockStream("abcd", j)

	x := New(stm)
	defer x.Done()

	ids := []DiscoIdentity{{
		Category: "server",
		Type:     "im",
		Name:     "default",
	}}
	x.SetIdentities(ids)

	fs := []DiscoFeature{
		discoInfoNamespace,
		discoItemsNamespace,
	}
	x.SetFeatures(fs)

	iq1 := xml.NewIQType(uuid.New(), xml.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(srvJid)
	iq1.AppendElement(xml.NewElementNamespace("query", discoInfoNamespace))

	x.ProcessIQ(iq1)
	elem := stm.FetchElement()
	require.NotNil(t, elem)
	q := elem.Elements().ChildNamespace("query", discoInfoNamespace)
	require.Equal(t, 3, q.Elements().Count())
	require.Equal(t, "identity", q.Elements().All()[0].Name())
	require.Equal(t, "feature", q.Elements().All()[1].Name())
}

func TestXEP0030_GetItems(t *testing.T) {
	srvJid, _ := xml.NewJID("", "jackal.im", "", true)

	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	stm := c2s.NewMockStream("abcd", j)

	x := New(stm)
	defer x.Done()

	its := []DiscoItem{
		{Jid: "j1@jackal.im", Name: "a name", Node: "node1"},
		{Jid: "j2@jackal.im", Name: "a second name", Node: "node2"},
	}
	x.SetItems(its)

	iq1 := xml.NewIQType(uuid.New(), xml.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(srvJid)
	iq1.AppendElement(xml.NewElementNamespace("query", discoItemsNamespace))

	x.ProcessIQ(iq1)
	elem := stm.FetchElement()
	require.NotNil(t, elem)
	q := elem.Elements().ChildNamespace("query", discoItemsNamespace)
	require.Equal(t, 2, q.Elements().Count())
	require.Equal(t, "item", q.Elements().All()[0].Name())
}
