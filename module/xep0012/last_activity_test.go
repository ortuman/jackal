/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0012

import (
	"context"
	"crypto/tls"
	"testing"

	c2srouter "github.com/ortuman/jackal/c2s/router"

	"github.com/ortuman/jackal/model"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/router"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0012_Matching(t *testing.T) {
	r, userRep, rosterRep := setupTest("jackal.im")

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	x := New(nil, r, userRep, rosterRep)
	defer func() { _ = x.Shutdown() }()

	// test MatchesIQ
	iq1 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq1.SetFromJID(j)

	require.False(t, x.MatchesIQ(iq1))

	iq1.AppendElement(xmpp.NewElementNamespace("query", lastActivityNamespace))

	iq2 := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq2.SetFromJID(j)
	iq2.AppendElement(xmpp.NewElementNamespace("query", lastActivityNamespace))

	require.True(t, x.MatchesIQ(iq1))
	require.True(t, x.MatchesIQ(iq2))

	iq1.SetType(xmpp.SetType)
	iq2.SetType(xmpp.ResultType)

	require.False(t, x.MatchesIQ(iq1))
	require.False(t, x.MatchesIQ(iq2))
}

func TestXEP0012_GetServerLastActivity(t *testing.T) {
	r, userRep, rosterRep := setupTest("jackal.im")

	j1, _ := jid.New("", "jackal.im", "", true)
	j2, _ := jid.New("ortuman", "jackal.im", "garden", true)

	stm := stream.NewMockC2S("abcd", j2)
	stm.SetPresence(xmpp.NewPresence(j2, j2, xmpp.AvailableType))

	defer stm.Disconnect(context.Background(), nil)

	x := New(nil, r, userRep, rosterRep)
	defer func() { _ = x.Shutdown() }()

	r.Bind(context.Background(), stm)

	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j1)
	iq.AppendElement(xmpp.NewElementNamespace("query", lastActivityNamespace))

	x.ProcessIQ(context.Background(), iq)
	elem := stm.ReceiveElement()
	q := elem.Elements().Child("query")
	require.NotNil(t, q)
	secs := q.Attributes().Get("seconds")
	require.True(t, len(secs) > 0)
}

func TestXEP0012_GetOnlineUserLastActivity(t *testing.T) {
	r, userRep, rosterRep := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "garden", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm1.SetPresence(xmpp.NewPresence(j1, j1, xmpp.AvailableType))

	stm2 := stream.NewMockC2S(uuid.New(), j2)
	stm2.SetPresence(xmpp.NewPresence(j2, j2, xmpp.AvailableType))

	x := New(nil, r, userRep, rosterRep)
	defer func() { _ = x.Shutdown() }()

	r.Bind(context.Background(), stm1)

	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j2.ToBareJID())
	iq.AppendElement(xmpp.NewElementNamespace("query", lastActivityNamespace))

	x.ProcessIQ(context.Background(), iq)
	elem := stm1.ReceiveElement()
	require.Equal(t, xmpp.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	p := xmpp.NewPresence(j1, j1, xmpp.UnavailableType)
	st := xmpp.NewElementName("status")
	st.SetText("Gone!")
	p.AppendElement(st)

	_ = userRep.UpsertUser(context.Background(), &model.User{
		Username:     "noelia",
		LastPresence: p,
	})
	_, _ = rosterRep.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})
	x.ProcessIQ(context.Background(), iq)
	elem = stm1.ReceiveElement()
	q := elem.Elements().ChildNamespace("query", lastActivityNamespace)
	secs := q.Attributes().Get("seconds")
	require.True(t, len(secs) > 0)

	// set as online
	r.Bind(context.Background(), stm2)

	x.ProcessIQ(context.Background(), iq)
	elem = stm1.ReceiveElement()
	q = elem.Elements().ChildNamespace("query", lastActivityNamespace)
	secs = q.Attributes().Get("seconds")
	require.Equal(t, "0", secs)

	memorystorage.EnableMockedError()
	x.ProcessIQ(context.Background(), iq)
	elem = stm1.ReceiveElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	memorystorage.DisableMockedError()
}

func setupTest(domain string) (router.Router, repository.User, repository.Roster) {
	userRep := memorystorage.NewUser()
	rosterRep := memorystorage.NewRoster()
	r, _ := router.New(
		&router.Config{
			Hosts: []router.HostConfig{{Name: domain, Certificate: tls.Certificate{}}},
		},
		c2srouter.New(userRep, memorystorage.NewBlockList()),
		nil,
	)
	return r, userRep, rosterRep
}
