/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"testing"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestRoster_MatchesIQ(t *testing.T) {
	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	stm := c2s.NewMockStream("abcd", j1)
	stm.SetUsername("ortuman")
	stm.SetDomain("jackal.im")

	r := NewRoster(&config.ModRoster{}, stm)
	defer r.Done()

	require.Equal(t, []string{}, r.AssociatedNamespaces())

	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.AppendElement(xml.NewElementNamespace("query", rosterNamespace))

	require.True(t, r.MatchesIQ(iq))
}

func TestRoster_FetchRoster(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	stm := c2s.NewMockStream("abcd", j1)
	stm.SetUsername("ortuman")
	stm.SetDomain("jackal.im")

	r := NewRoster(&config.ModRoster{}, stm)

	iq := xml.NewIQType(uuid.New(), xml.ResultType)
	q := xml.NewElementNamespace("query", rosterNamespace)
	q.AppendElement(xml.NewElementName("q2"))
	iq.AppendElement(q)

	r.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xml.GetType)
	r.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())
	q.ClearElements()

	r.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.ResultType, elem.Type())

	query := elem.Elements().ChildNamespace("query", rosterNamespace)
	require.Equal(t, 0, query.Elements().Count())
	r.Done()

	ri1 := &model.RosterItem{
		User:         "ortuman",
		Contact:      "noelia",
		Name:         "My Juliet",
		Subscription: subscriptionNone,
		Ask:          true,
		Groups:       []string{"people", "friends"},
	}
	storage.Instance().InsertOrUpdateRosterItem(ri1)
	ri2 := &model.RosterItem{
		User:         "ortuman",
		Contact:      "romeo",
		Name:         "Rome",
		Subscription: subscriptionNone,
		Ask:          true,
		Groups:       []string{"others"},
	}
	storage.Instance().InsertOrUpdateRosterItem(ri2)

	r = NewRoster(&config.ModRoster{Versioning: true}, stm)
	r.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.ResultType, elem.Type())

	query2 := elem.Elements().ChildNamespace("query", rosterNamespace)
	require.Equal(t, 2, query2.Elements().Count())
	require.True(t, r.IsRequested())

	// test versioning
	iq = xml.NewIQType(uuid.New(), xml.GetType)
	q = xml.NewElementNamespace("query", rosterNamespace)
	q.SetAttribute("ver", "v1")
	iq.AppendElement(q)

	r.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.ResultType, elem.Type())

	// expect set item...
	elem = stm.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	query2 = elem.Elements().ChildNamespace("query", rosterNamespace)
	require.Equal(t, "v2", query2.Attributes().Get("ver"))
	item := query2.Elements().Child("item")
	require.Equal(t, "romeo@jackal.im", item.Attributes().Get("jid"))
	r.Done()

	storage.ActivateMockedError()
	r = NewRoster(&config.ModRoster{}, stm)
	r.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	r.Done()
	storage.DeactivateMockedError()
}

func TestRoster_DeliverPendingApprovalNotifications(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer c2s.Shutdown()

	rn := model.RosterNotification{
		User:     "noelia",
		Contact:  "ortuman",
		Elements: []xml.XElement{xml.NewElementName("group")},
	}
	storage.Instance().InsertOrUpdateRosterNotification(&rn)

	stm, _ := tUtilRosterInitializeRoster()

	r := NewRoster(&config.ModRoster{}, stm)
	defer r.Done()

	storage.ActivateMockedError()
	ch := make(chan bool)
	r.errHandler = func(error) {
		close(ch)
	}
	r.DeliverPendingApprovalNotifications()
	<-ch
	storage.DeactivateMockedError()

	r.DeliverPendingApprovalNotifications()
	elem := stm.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.SubscribeType, elem.Type())
	require.Equal(t, "noelia@jackal.im", elem.From())
	require.NotNil(t, elem.Elements().Child("group"))
}

func TestRoster_ReceiveAndBroadcastPresence(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer c2s.Shutdown()

	stm1, stm2 := tUtilRosterInitializeRoster()

	// insert roster item...
	ri := &model.RosterItem{
		User:         "ortuman",
		Contact:      "noelia",
		Name:         "My Juliet",
		Subscription: subscriptionBoth,
	}
	storage.Instance().InsertOrUpdateRosterItem(ri)

	r := NewRoster(&config.ModRoster{}, stm1)
	defer r.Done()

	// test presence receive...
	storage.ActivateMockedError()
	ch := make(chan bool)
	r.errHandler = func(error) {
		close(ch)
	}
	r.ReceivePresences()
	<-ch
	storage.DeactivateMockedError()

	r.ReceivePresences()
	elem := stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, "available", elem.Type())
	require.Equal(t, "noelia@jackal.im/garden", elem.From())

	// test broadcast presence...
	presence := xml.NewPresence(stm1.JID(), stm2.JID().ToBareJID(), xml.AvailableType)

	storage.ActivateMockedError()
	ch2 := make(chan bool)
	r.errHandler = func(error) {
		close(ch2)
	}
	r.BroadcastPresence(presence)
	<-ch2
	ch3 := make(chan bool)
	r.errHandler = func(error) {
		close(ch3)
	}
	r.BroadcastPresenceAndWait(presence)
	<-ch3
	storage.DeactivateMockedError()

	r.BroadcastPresence(presence)
	elem = stm2.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, "available", elem.Type())
	require.Equal(t, "ortuman@jackal.im/balcony", elem.From())

	r.BroadcastPresenceAndWait(presence)
	elem = stm2.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, "available", elem.Type())
	require.Equal(t, "ortuman@jackal.im/balcony", elem.From())
}

func TestRoster_Update(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer c2s.Shutdown()

	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	stm1 := c2s.NewMockStream("abcd1234", j1)
	stm1.SetUsername("ortuman")
	stm1.SetDomain("jackal.im")
	stm1.SetResource("garden")
	stm1.SetAuthenticated(true)

	r := NewRoster(&config.ModRoster{}, stm1)
	defer r.Done()

	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	q := xml.NewElementNamespace("query", rosterNamespace)
	item := xml.NewElementName("item")
	item.SetAttribute("jid", "noelia@jackal.im")
	item.SetAttribute("subscription", subscriptionNone)
	item.SetAttribute("name", "My Juliet")
	q.AppendElement(item)
	q.AppendElement(item)
	iq.AppendElement(q)

	r.ProcessIQ(iq)
	elem := stm1.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	q.ClearElements()
	q.AppendElement(item)

	r.ProcessIQ(iq)
	elem = stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	ri, err := storage.Instance().FetchRosterItem("ortuman", "noelia")
	require.Nil(t, err)
	require.NotNil(t, ri)
	require.Equal(t, "ortuman", ri.User)
	require.Equal(t, "noelia", ri.Contact)
}

func TestRoster_Subscribe(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer c2s.Shutdown()

	stm1, stm2 := tUtilRosterInitializeRoster()

	r := NewRoster(&config.ModRoster{}, stm1)
	defer r.Done()

	tUtilRosterRequestRoster(r, stm1)

	// send subscribe presence...
	presence := xml.NewPresence(stm1.JID(), stm2.JID().ToBareJID(), xml.SubscribeType)

	r.ProcessPresence(presence)
	elem := stm2.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, "subscribe", elem.Type())
	require.Equal(t, "ortuman@jackal.im", elem.From())
}

func TestRoster_Subscribed(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer c2s.Shutdown()

	// insert roster item...
	ri := &model.RosterItem{
		User:         "ortuman",
		Contact:      "noelia",
		Name:         "My Juliet",
		Subscription: subscriptionNone,
	}
	storage.Instance().InsertOrUpdateRosterItem(ri)

	// insert roster approval notification...
	rn := &model.RosterNotification{
		User:     "ortuman",
		Contact:  "noelia",
		Elements: []xml.XElement{},
	}
	storage.Instance().InsertOrUpdateRosterNotification(rn)

	stm1, stm2 := tUtilRosterInitializeRoster()

	r1 := NewRoster(&config.ModRoster{}, stm1)
	r2 := NewRoster(&config.ModRoster{}, stm2)
	defer r1.Done()
	defer r2.Done()

	tUtilRosterRequestRoster(r1, stm1)
	tUtilRosterRequestRoster(r2, stm2)

	// send subscribe presence...
	presence := xml.NewPresence(stm2.JID(), stm1.JID().ToBareJID(), xml.SubscribedType)

	r2.ProcessPresence(presence)
	elem := stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	require.NotNil(t, elem.Elements().ChildNamespace("query", rosterNamespace))

	elem = stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.SubscribedType, elem.Type())

	elem = stm2.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	qRes := elem.Elements().ChildNamespace("query", rosterNamespace)
	require.NotNil(t, qRes)
	iRes := qRes.Elements().Child("item")
	require.Equal(t, subscriptionFrom, iRes.Attributes().Get("subscription"))

	rns, err := storage.Instance().FetchRosterNotifications("noelia")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))

	ri, err = storage.Instance().FetchRosterItem("ortuman", "noelia")
	require.Nil(t, err)
	require.Equal(t, subscriptionTo, ri.Subscription)
}

func TestRoster_Unsubscribe(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer c2s.Shutdown()

	tUtilRosterInsertRosterItems()
	stm1, stm2 := tUtilRosterInitializeRoster()

	r1 := NewRoster(&config.ModRoster{}, stm1)
	r2 := NewRoster(&config.ModRoster{}, stm2)
	defer r1.Done()
	defer r2.Done()

	tUtilRosterRequestRoster(r1, stm1)
	tUtilRosterRequestRoster(r2, stm2)

	// send unsubscribe presence...
	presence := xml.NewPresence(stm1.JID(), stm2.JID().ToBareJID(), xml.UnsubscribeType)

	r1.ProcessPresence(presence)
	elem := stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("query", rosterNamespace))

	elem = stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.UnavailableType, elem.Type())

	elem = stm2.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	qRes := elem.Elements().ChildNamespace("query", rosterNamespace)
	require.NotNil(t, qRes)
	iRes := qRes.Elements().Child("item")
	require.Equal(t, subscriptionTo, iRes.Attributes().Get("subscription"))

	elem = stm2.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.UnsubscribeType, elem.Type())
}

func TestRoster_Unsubscribed(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer c2s.Shutdown()

	tUtilRosterInsertRosterItems()
	stm1, stm2 := tUtilRosterInitializeRoster()

	r1 := NewRoster(&config.ModRoster{}, stm1)
	r2 := NewRoster(&config.ModRoster{}, stm2)
	defer r1.Done()
	defer r2.Done()

	tUtilRosterRequestRoster(r1, stm1)
	tUtilRosterRequestRoster(r2, stm2)

	// send unsubscribed presence...
	presence := xml.NewPresence(stm2.JID(), stm1.JID().ToBareJID(), xml.UnsubscribedType)

	r2.ProcessPresence(presence)
	elem := stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	qRes := elem.Elements().ChildNamespace("query", rosterNamespace)
	require.NotNil(t, qRes)
	iRes := qRes.Elements().Child("item")
	require.Equal(t, subscriptionFrom, iRes.Attributes().Get("subscription"))

	elem = stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.UnsubscribedType, elem.Type())

	elem = stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.UnavailableType, elem.Type())
	require.Equal(t, "noelia@jackal.im/garden", elem.From())

	elem = stm2.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	require.NotNil(t, elem.Elements().ChildNamespace("query", rosterNamespace))
}

func TestRoster_DeleteItem(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer c2s.Shutdown()

	tUtilRosterInsertRosterItems()
	stm1, stm2 := tUtilRosterInitializeRoster()

	r1 := NewRoster(&config.ModRoster{}, stm1)
	r2 := NewRoster(&config.ModRoster{}, stm2)
	defer r1.Done()
	defer r2.Done()

	tUtilRosterRequestRoster(r1, stm1)
	tUtilRosterRequestRoster(r2, stm2)

	// delete item IQ...
	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	item := xml.NewElementName("item")
	item.SetAttribute("jid", "noelia@jackal.im")
	item.SetAttribute("subscription", subscriptionRemove)
	q := xml.NewElementNamespace("query", rosterNamespace)
	q.AppendElement(item)
	iq.AppendElement(q)

	r1.ProcessIQ(iq)
	elem := stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	qRes := elem.Elements().ChildNamespace("query", rosterNamespace)
	require.NotNil(t, qRes)
	iRes := qRes.Elements().Child("item")
	require.Equal(t, subscriptionRemove, iRes.Attributes().Get("subscription"))

	elem = stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.UnavailableType, elem.Type())
	require.Equal(t, "noelia@jackal.im/garden", elem.From())

	elem = stm1.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	elem = stm2.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	qRes = elem.Elements().ChildNamespace("query", rosterNamespace)
	require.NotNil(t, qRes)
	iRes = qRes.Elements().Child("item")
	require.Equal(t, subscriptionTo, iRes.Attributes().Get("subscription"))

	elem = stm2.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	qRes = elem.Elements().ChildNamespace("query", rosterNamespace)
	require.NotNil(t, qRes)
	iRes = qRes.Elements().Child("item")
	require.Equal(t, subscriptionNone, iRes.Attributes().Get("subscription"))

	elem = stm2.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.UnsubscribeType, elem.Type())
	require.Equal(t, "ortuman@jackal.im", elem.From())
}

func tUtilRosterInsertRosterItems() {
	// insert roster item...
	ri1 := &model.RosterItem{
		User:         "ortuman",
		Contact:      "noelia",
		Name:         "My Juliet",
		Subscription: subscriptionBoth,
	}
	ri2 := &model.RosterItem{
		User:         "noelia",
		Contact:      "ortuman",
		Name:         "My Romeo",
		Subscription: subscriptionBoth,
	}
	storage.Instance().InsertOrUpdateRosterItem(ri1)
	storage.Instance().InsertOrUpdateRosterItem(ri2)
}

func tUtilRosterRequestRoster(r *ModRoster, stm *c2s.MockStream) {
	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.AppendElement(xml.NewElementNamespace("query", rosterNamespace))

	r.ProcessIQ(iq)
	_ = stm.FetchElement()
}

func tUtilRosterInitializeRoster() (*c2s.MockStream, *c2s.MockStream) {
	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	j2, _ := xml.NewJID("noelia", "jackal.im", "garden", true)

	stm1 := c2s.NewMockStream("abcd1234", j1)
	stm1.SetUsername("ortuman")
	stm1.SetDomain("jackal.im")
	stm1.SetResource("garden")
	stm1.SetAuthenticated(true)
	stm1.SetRosterRequested(true)
	stm1.SetJID(j1)

	stm2 := c2s.NewMockStream("abcd5678", j2)
	stm2.SetUsername("noelia")
	stm2.SetDomain("jackal.im")
	stm2.SetResource("garden")
	stm2.SetAuthenticated(true)
	stm2.SetRosterRequested(true)
	stm2.SetJID(j2)

	// register streams...
	c2s.Instance().RegisterStream(stm1)
	c2s.Instance().RegisterStream(stm2)
	c2s.Instance().AuthenticateStream(stm1)
	c2s.Instance().AuthenticateStream(stm2)

	return stm1, stm2
}
