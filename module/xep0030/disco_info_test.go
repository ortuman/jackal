/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0030

import (
	"context"
	"crypto/tls"
	"testing"

	c2srouter "github.com/ortuman/jackal/c2s/router"

	"github.com/ortuman/jackal/module/xep0004"
	"github.com/ortuman/jackal/router"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0030_Matching(t *testing.T) {
	r, rosterRep := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	x := New(r, rosterRep)
	defer func() { _ = x.Shutdown() }()

	// test MatchesIQ
	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j.ToBareJID())

	require.False(t, x.MatchesIQ(iq1))

	iq1.AppendElement(xmpp.NewElementNamespace("query", discoItemsNamespace))

	iq2 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq2.SetFromJID(j)
	iq2.SetToJID(j.ToBareJID())
	iq2.AppendElement(xmpp.NewElementNamespace("query", discoItemsNamespace))

	require.True(t, x.MatchesIQ(iq1))
	require.True(t, x.MatchesIQ(iq2))

	iq1.SetType(xmpp.SetType)
	iq2.SetType(xmpp.ResultType)

	require.False(t, x.MatchesIQ(iq1))
	require.False(t, x.MatchesIQ(iq2))
}

func TestXEP0030_SendFeatures(t *testing.T) {
	r, rosterRep := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	srvJid, _ := jid.New("", "jackal.im", "", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	stm.SetPresence(xmpp.NewPresence(j, j, xmpp.AvailableType))

	r.Bind(context.Background(), stm)

	x := New(r, rosterRep)
	defer func() { _ = x.Shutdown() }()

	x.RegisterServerFeature("s0")
	x.RegisterServerFeature("s1")
	x.RegisterServerFeature("s2")
	x.RegisterAccountFeature("af0")
	x.RegisterAccountFeature("af1")

	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(srvJid)
	iq1.AppendElement(xmpp.NewElementNamespace("query", discoInfoNamespace))

	x.ProcessIQ(context.Background(), iq1)
	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	q := elem.Elements().ChildNamespace("query", discoInfoNamespace)

	require.NotNil(t, q)
	require.Equal(t, 6, q.Elements().Count())
	require.Equal(t, "identity", q.Elements().All()[0].Name())
	require.Equal(t, "feature", q.Elements().All()[1].Name())

	x.UnregisterServerFeature("s1")
	x.UnregisterAccountFeature("af1")

	x.ProcessIQ(context.Background(), iq1)
	elem = stm.ReceiveElement()
	q = elem.Elements().ChildNamespace("query", discoInfoNamespace)

	require.NotNil(t, q)
	require.Equal(t, 5, q.Elements().Count())

	iq1.SetToJID(j.ToBareJID())
	x.ProcessIQ(context.Background(), iq1)
	elem = stm.ReceiveElement()
	q = elem.Elements().ChildNamespace("query", discoInfoNamespace)

	require.NotNil(t, q)
	require.Equal(t, 4, q.Elements().Count())
}

func TestXEP0030_SendItems(t *testing.T) {
	r, rosterRep := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	stm.SetPresence(xmpp.NewPresence(j, j, xmpp.AvailableType))

	r.Bind(context.Background(), stm)

	x := New(r, rosterRep)
	defer func() { _ = x.Shutdown() }()

	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j.ToBareJID())
	iq1.AppendElement(xmpp.NewElementNamespace("query", discoItemsNamespace))

	x.ProcessIQ(context.Background(), iq1)
	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	q := elem.Elements().ChildNamespace("query", discoItemsNamespace)

	require.Equal(t, 1, len(q.Elements().Children("item")))
}

type testDiscoInfoProvider struct {
}

func (tp *testDiscoInfoProvider) Identities(_ context.Context, toJID, fromJID *jid.JID, node string) []Identity {
	return []Identity{{Name: "test_identity"}}
}

func (tp *testDiscoInfoProvider) Items(_ context.Context, toJID, fromJID *jid.JID, node string) ([]Item, *xmpp.StanzaError) {
	return []Item{{Jid: "test.jackal.im"}}, nil
}

func (tp *testDiscoInfoProvider) Features(_ context.Context, toJID, fromJID *jid.JID, node string) ([]Feature, *xmpp.StanzaError) {
	return []Feature{"com.jackal.im.feature"}, nil
}

func (tp *testDiscoInfoProvider) Form(_ context.Context, toJID, fromJID *jid.JID, node string) (*xep0004.DataForm, *xmpp.StanzaError) {
	return nil, nil
}

func TestXEP0030_Provider(t *testing.T) {
	r, rosterRep := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	compJID, _ := jid.New("", "test.jackal.im", "", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	stm.SetPresence(xmpp.NewPresence(j, j, xmpp.AvailableType))

	r.Bind(context.Background(), stm)

	x := New(r, rosterRep)
	defer func() { _ = x.Shutdown() }()

	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(compJID)
	iq1.AppendElement(xmpp.NewElementNamespace("query", discoItemsNamespace))

	x.ProcessIQ(context.Background(), iq1)
	elem := stm.ReceiveElement()
	require.True(t, elem.IsError())
	require.Equal(t, xmpp.ErrItemNotFound.Error(), elem.Error().Elements().All()[0].Name())

	x.RegisterProvider(compJID.String(), &testDiscoInfoProvider{})

	x.ProcessIQ(context.Background(), iq1)
	elem = stm.ReceiveElement()
	q := elem.Elements().ChildNamespace("query", discoItemsNamespace)
	require.NotNil(t, q)

	require.Equal(t, 1, len(q.Elements().Children("item")))

	x.UnregisterProvider(compJID.String())

	x.ProcessIQ(context.Background(), iq1)
	elem = stm.ReceiveElement()
	require.True(t, elem.IsError())
	require.Equal(t, xmpp.ErrItemNotFound.Error(), elem.Error().Elements().All()[0].Name())
}

func setupTest(domain string) (router.Router, repository.Roster) {
	rosterRep := memorystorage.NewRoster()
	r, _ := router.New(
		&router.Config{
			Hosts: []router.HostConfig{{Name: domain, Certificate: tls.Certificate{}}},
		},
		c2srouter.New(memorystorage.NewUser(), memorystorage.NewBlockList()),
	)
	return r, rosterRep
}
