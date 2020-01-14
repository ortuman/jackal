/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/ortuman/jackal/model"
	rostermodel "github.com/ortuman/jackal/model/roster"
	"github.com/ortuman/jackal/module/presencehub"
	"github.com/ortuman/jackal/router"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/storage/repository"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestRoster_MatchesIQ(t *testing.T) {
	rtr, userRep, rosterRep := setupTest("jackal.im")

	r := New(&Config{}, presencehub.New(rtr, nil), nil, rtr, userRep, rosterRep)
	defer func() { _ = r.Shutdown() }()

	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.AppendElement(xmpp.NewElementNamespace("query", rosterNamespace))

	require.True(t, r.MatchesIQ(iq))
}

func TestRoster_FetchRoster(t *testing.T) {
	rtr, userRep, rosterRep := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	rtr.Bind(context.Background(), stm)

	r := New(&Config{}, presencehub.New(rtr, nil), nil, rtr, userRep, rosterRep)
	defer func() { _ = r.Shutdown() }()

	iq := xmpp.NewIQType(uuid.New(), xmpp.ResultType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1.ToBareJID())
	q := xmpp.NewElementNamespace("query", rosterNamespace)
	q.AppendElement(xmpp.NewElementName("q2"))
	iq.AppendElement(q)

	r.ProcessIQ(context.Background(), iq)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xmpp.GetType)
	r.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())
	q.ClearElements()

	r.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	query := elem.Elements().ChildNamespace("query", rosterNamespace)
	require.Equal(t, 0, query.Elements().Count())

	ri1 := &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Name:         "My Juliet",
		Subscription: rostermodel.SubscriptionNone,
		Ask:          true,
		Groups:       []string{"people", "friends"},
	}
	_, _ = rosterRep.UpsertRosterItem(context.Background(), ri1)

	ri2 := &rostermodel.Item{
		Username:     "ortuman",
		JID:          "romeo@jackal.im",
		Name:         "Rome",
		Subscription: rostermodel.SubscriptionNone,
		Ask:          true,
		Groups:       []string{"others"},
	}
	_, _ = rosterRep.UpsertRosterItem(context.Background(), ri2)

	r = New(&Config{Versioning: true}, presencehub.New(rtr, nil), nil, rtr, userRep, rosterRep)
	defer func() { _ = r.Shutdown() }()

	r.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	query2 := elem.Elements().ChildNamespace("query", rosterNamespace)
	require.Equal(t, 2, query2.Elements().Count())
	require.True(t, stm.GetBool(rosterRequestedCtxKey))

	// test versioning
	iq = xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1.ToBareJID())
	q = xmpp.NewElementNamespace("query", rosterNamespace)
	q.SetAttribute("ver", "v1")
	iq.AppendElement(q)

	r.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())

	// expect set item...
	elem = stm.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.SetType, elem.Type())
	query2 = elem.Elements().ChildNamespace("query", rosterNamespace)
	require.Equal(t, "v2", query2.Attributes().Get("ver"))
	item := query2.Elements().Child("item")
	require.Equal(t, "romeo@jackal.im", item.Attributes().Get("jid"))

	memorystorage.EnableMockedError()
	r = New(&Config{}, presencehub.New(rtr, nil), nil, rtr, userRep, rosterRep)
	defer func() { _ = r.Shutdown() }()

	r.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	memorystorage.DisableMockedError()
}

func TestRoster_Update(t *testing.T) {
	rtr, userRep, rosterRep := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "garden", true)
	j2, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm1.SetAuthenticated(true)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	stm2.SetAuthenticated(true)
	stm2.SetBool(context.Background(), rosterRequestedCtxKey, true)

	r := New(&Config{}, presencehub.New(rtr, nil), nil, rtr, userRep, rosterRep)
	defer func() { _ = r.Shutdown() }()

	rtr.Bind(context.Background(), stm1)
	rtr.Bind(context.Background(), stm2)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j1.ToBareJID())
	q := xmpp.NewElementNamespace("query", rosterNamespace)
	item := xmpp.NewElementName("item")
	item.SetAttribute("jid", "noelia@jackal.im")
	item.SetAttribute("subscription", rostermodel.SubscriptionNone)
	item.SetAttribute("name", "My Juliet")
	q.AppendElement(item)
	q.AppendElement(item)
	iq.AppendElement(q)

	r.ProcessIQ(context.Background(), iq)
	elem := stm1.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	q.ClearElements()
	q.AppendElement(item)

	r.ProcessIQ(context.Background(), iq)
	elem = stm1.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	// expecting roster push...
	elem = stm2.ReceiveElement()
	require.Equal(t, xmpp.SetType, elem.Type())

	// update name
	item.SetAttribute("name", "My Girl")
	q.ClearElements()
	q.AppendElement(item)

	r.ProcessIQ(context.Background(), iq)
	elem = stm1.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xmpp.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	ri, err := rosterRep.FetchRosterItem(context.Background(), "ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.NotNil(t, ri)
	require.Equal(t, "ortuman", ri.Username)
	require.Equal(t, "noelia@jackal.im", ri.JID)
	require.Equal(t, "My Girl", ri.Name)
}

func TestRoster_RemoveItem(t *testing.T) {
	rtr, userRep, rosterRep := setupTest("jackal.im")

	// insert contact's roster item
	_, _ = rosterRep.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Name:         "My Juliet",
		Subscription: rostermodel.SubscriptionBoth,
	})
	_, _ = rosterRep.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "noelia",
		JID:          "ortuman@jackal.im",
		Name:         "My Romeo",
		Subscription: rostermodel.SubscriptionBoth,
	})
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	rtr.Bind(context.Background(), stm)

	r := New(&Config{}, presencehub.New(rtr, nil), nil, rtr, userRep, rosterRep)
	defer func() { _ = r.Shutdown() }()

	// remove item
	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())

	q := xmpp.NewElementNamespace("query", rosterNamespace)
	item := xmpp.NewElementName("item")
	item.SetAttribute("jid", "noelia@jackal.im")
	item.SetAttribute("subscription", rostermodel.SubscriptionRemove)
	q.AppendElement(item)
	iq.AppendElement(q)

	r.ProcessIQ(context.Background(), iq)
	elem := stm.ReceiveElement()
	require.Equal(t, iqID, elem.ID())

	ri, err := rosterRep.FetchRosterItem(context.Background(), "ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Nil(t, ri)
}

func TestRoster_OnlineJIDs(t *testing.T) {
	rtr, userRep, rosterRep := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "garden", true)
	j3, _ := jid.New("fruela", "jackal.im", "balcony", true)
	j4, _ := jid.New("ortuman", "jackal.im", "yard", true)
	j5, _ := jid.New("boss", "jabber.org", "yard", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm1.SetAuthenticated(true)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	stm2.SetAuthenticated(true)

	rtr.Bind(context.Background(), stm1)
	rtr.Bind(context.Background(), stm2)

	// user entity
	_ = userRep.UpsertUser(context.Background(), &model.User{
		Username:     "ortuman",
		LastPresence: xmpp.NewPresence(j1, j1.ToBareJID(), xmpp.UnavailableType),
	})

	// roster items
	_, _ = rosterRep.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "noelia",
		JID:          "ortuman@jackal.im",
		Subscription: rostermodel.SubscriptionBoth,
	})
	_, _ = rosterRep.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: rostermodel.SubscriptionBoth,
	})

	// pending notification
	_ = rosterRep.UpsertRosterNotification(context.Background(), &rostermodel.Notification{
		Contact:  "ortuman",
		JID:      j3.ToBareJID().String(),
		Presence: xmpp.NewPresence(j3.ToBareJID(), j1.ToBareJID(), xmpp.SubscribeType),
	})

	ph := presencehub.New(rtr, nil)
	r := New(&Config{}, ph, nil, rtr, userRep, rosterRep)
	defer func() { _ = r.Shutdown() }()

	// online presence...
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j1, j1.ToBareJID(), xmpp.AvailableType))

	time.Sleep(time.Millisecond * 150) // wait until processed...

	// receive pending approval notification...
	elem := stm1.ReceiveElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, j3.ToBareJID().String(), elem.From())
	require.Equal(t, xmpp.SubscribeType, elem.Type())

	// expect user's available presence
	elem = stm2.ReceiveElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, j1.String(), elem.From())
	require.Equal(t, xmpp.AvailableType, elem.Type())

	// check if last presence was updated
	usr, err := userRep.FetchUser(context.Background(), "ortuman")
	require.Nil(t, err)
	require.NotNil(t, usr)
	require.NotNil(t, usr.LastPresence)
	require.Equal(t, xmpp.AvailableType, usr.LastPresence.Type())

	// send remaining online presences...
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j2, j2.ToBareJID(), xmpp.AvailableType))
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j3, j3.ToBareJID(), xmpp.AvailableType))
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j4, j1.ToBareJID(), xmpp.AvailableType))
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j5, j1.ToBareJID(), xmpp.AvailableType))

	time.Sleep(time.Millisecond * 150) // wait until processed...

	require.Equal(t, 1, len(ph.AvailablePresencesMatchingJID(j1)))

	j6, _ := jid.NewWithString("jackal.im", true)
	require.Equal(t, 4, len(ph.AvailablePresencesMatchingJID(j6)))

	j7, _ := jid.NewWithString("jabber.org", true)
	require.Equal(t, 1, len(ph.AvailablePresencesMatchingJID(j7)))

	j8, _ := jid.NewWithString("jackal.im/balcony", true)
	require.Equal(t, 2, len(ph.AvailablePresencesMatchingJID(j8)))

	j9, _ := jid.NewWithString("ortuman@jackal.im", true)
	require.Equal(t, 2, len(ph.AvailablePresencesMatchingJID(j9)))

	// send unavailable presences...
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j1, j1.ToBareJID(), xmpp.UnavailableType))
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j2, j2.ToBareJID(), xmpp.UnavailableType))
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j3, j3.ToBareJID(), xmpp.UnavailableType))
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j4, j4.ToBareJID(), xmpp.UnavailableType))
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j5, j1.ToBareJID(), xmpp.UnavailableType))

	time.Sleep(time.Millisecond * 150) // wait until processed...

	require.Equal(t, 0, len(ph.AvailablePresencesMatchingJID(j1)))
	require.Equal(t, 0, len(ph.AvailablePresencesMatchingJID(j6)))
	require.Equal(t, 0, len(ph.AvailablePresencesMatchingJID(j7)))
	require.Equal(t, 0, len(ph.AvailablePresencesMatchingJID(j8)))
	require.Equal(t, 0, len(ph.AvailablePresencesMatchingJID(j9)))
}

func TestRoster_Probe(t *testing.T) {
	rtr, userRep, rosterRep := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "garden", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	stm.SetAuthenticated(true)

	rtr.Bind(context.Background(), stm)

	r := New(&Config{}, presencehub.New(rtr, nil), nil, rtr, userRep, rosterRep)
	defer func() { _ = r.Shutdown() }()

	_ = userRep.UpsertUser(context.Background(), &model.User{
		Username:     "noelia",
		LastPresence: xmpp.NewPresence(j2.ToBareJID(), j2.ToBareJID(), xmpp.UnavailableType),
	})

	_, _ = rosterRep.UpsertRosterItem(context.Background(), &rostermodel.Item{
		Username:     "noelia",
		JID:          "ortuman@jackal.im",
		Subscription: rostermodel.SubscriptionFrom,
	})
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j1, j2, xmpp.ProbeType))
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.UnavailableType, elem.Type())

	// test available presence...
	p2 := xmpp.NewPresence(j2, j2.ToBareJID(), xmpp.AvailableType)
	_ = userRep.UpsertUser(context.Background(), &model.User{
		Username:     "noelia",
		LastPresence: p2,
	})
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j1, j2, xmpp.ProbeType))
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.AvailableType, elem.Type())
	require.Equal(t, "noelia@jackal.im/garden", elem.From())
}

func TestRoster_Subscription(t *testing.T) {
	rtr, userRep, rosterRep := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "garden", true)

	r := New(&Config{}, presencehub.New(rtr, nil), nil, rtr, userRep, rosterRep)
	defer func() { _ = r.Shutdown() }()

	r.ProcessPresence(context.Background(), xmpp.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xmpp.SubscribeType))
	time.Sleep(time.Millisecond * 150) // wait until processed...

	rns, err := rosterRep.FetchRosterNotifications(context.Background(), "noelia")
	require.Nil(t, err)
	require.Equal(t, 1, len(rns))

	// resend request...
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xmpp.SubscribeType))

	// contact request cancellation
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j2.ToBareJID(), j1.ToBareJID(), xmpp.UnsubscribedType))
	time.Sleep(time.Millisecond * 150) // wait until processed...

	rns, err = rosterRep.FetchRosterNotifications(context.Background(), "noelia")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))

	ri, err := rosterRep.FetchRosterItem(context.Background(), "ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionNone, ri.Subscription)

	// contact accepts request...
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xmpp.SubscribeType))
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j2.ToBareJID(), j1.ToBareJID(), xmpp.SubscribedType))
	time.Sleep(time.Millisecond * 150) // wait until processed...

	ri, err = rosterRep.FetchRosterItem(context.Background(), "ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionTo, ri.Subscription)

	// contact subscribes to user's presence...
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j2.ToBareJID(), j1.ToBareJID(), xmpp.SubscribeType))
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xmpp.SubscribedType))
	time.Sleep(time.Millisecond * 150) // wait until processed...

	ri, err = rosterRep.FetchRosterItem(context.Background(), "noelia", "ortuman@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionBoth, ri.Subscription)

	// user unsubscribes from contact's presence...
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xmpp.UnsubscribeType))
	time.Sleep(time.Millisecond * 150) // wait until processed...

	ri, err = rosterRep.FetchRosterItem(context.Background(), "ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionFrom, ri.Subscription)

	// user cancels contact subscription
	r.ProcessPresence(context.Background(), xmpp.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xmpp.UnsubscribedType))
	time.Sleep(time.Millisecond * 150) // wait until processed...

	ri, err = rosterRep.FetchRosterItem(context.Background(), "ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionNone, ri.Subscription)

	ri, err = rosterRep.FetchRosterItem(context.Background(), "noelia", "ortuman@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionNone, ri.Subscription)
}

func setupTest(domain string) (*router.Router, repository.User, repository.Roster) {
	userRep := memorystorage.NewUser()
	rosterRep := memorystorage.NewRoster()
	r, _ := router.New(
		&router.Config{
			Hosts: []router.HostConfig{{Name: domain, Certificate: tls.Certificate{}}},
		},
		userRep,
		memorystorage.NewBlockList(),
	)
	return r, userRep, rosterRep
}
