/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

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

func TestRoster_MatchesIQ(t *testing.T) {
	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd", j1)
	stm.SetUsername("ortuman")
	stm.SetDomain("jackal.im")

	r := New(&Config{}, stm)
	defer stm.Disconnect(nil)

	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.AppendElement(xml.NewElementNamespace("query", rosterNamespace))

	require.True(t, r.MatchesIQ(iq))
}

func TestRoster_FetchRoster(t *testing.T) {
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer storage.Shutdown()

	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd", j1)
	stm.SetUsername("ortuman")
	stm.SetDomain("jackal.im")

	r := New(&Config{}, stm)
	defer stm.Disconnect(nil)

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

	ri1 := &model.RosterItem{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Name:         "My Juliet",
		Subscription: SubscriptionNone,
		Ask:          true,
		Groups:       []string{"people", "friends"},
	}
	storage.Instance().InsertOrUpdateRosterItem(ri1)

	ri2 := &model.RosterItem{
		Username:     "ortuman",
		JID:          "romeo@jackal.im",
		Name:         "Rome",
		Subscription: SubscriptionNone,
		Ask:          true,
		Groups:       []string{"others"},
	}
	storage.Instance().InsertOrUpdateRosterItem(ri2)

	r = New(&Config{Versioning: true}, stm)
	r.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.ResultType, elem.Type())

	query2 := elem.Elements().ChildNamespace("query", rosterNamespace)
	require.Equal(t, 2, query2.Elements().Count())
	require.True(t, stm.Context().Bool(rosterRequestedCtxKey))

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

	storage.ActivateMockedError()
	r = New(&Config{}, stm)
	r.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())

	storage.DeactivateMockedError()
}

func TestRoster_DeliverPendingApprovalNotifications(t *testing.T) {
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	rn := model.RosterNotification{
		Contact:  "ortuman",
		JID:      "noelia@jackal.im",
		Elements: []xml.XElement{xml.NewElementName("group")},
	}
	storage.Instance().InsertOrUpdateRosterNotification(&rn)

	stm, _ := tUtilRosterInitializeRoster()

	r := New(&Config{}, stm)

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
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	stm1, stm2 := tUtilRosterInitializeRoster()

	// insert roster item...
	ri := &model.RosterItem{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Name:         "My Juliet",
		Subscription: SubscriptionBoth,
	}
	storage.Instance().InsertOrUpdateRosterItem(ri)

	r := New(&Config{}, stm1)

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
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S("abcd1234", j1)
	stm1.SetUsername("ortuman")
	stm1.SetDomain("jackal.im")
	stm1.SetResource("garden")
	stm1.SetAuthenticated(true)

	r := New(&Config{}, stm1)

	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	q := xml.NewElementNamespace("query", rosterNamespace)
	item := xml.NewElementName("item")
	item.SetAttribute("jid", "noelia@jackal.im")
	item.SetAttribute("subscription", SubscriptionNone)
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

	ri, err := storage.Instance().FetchRosterItem("ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.NotNil(t, ri)
	require.Equal(t, "ortuman", ri.Username)
	require.Equal(t, "noelia@jackal.im", ri.JID)
}

func TestRoster_Subscribe(t *testing.T) {
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	stm1, stm2 := tUtilRosterInitializeRoster()

	r := New(&Config{}, stm1)

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
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	// insert roster item...
	ri := &model.RosterItem{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Name:         "My Juliet",
		Subscription: SubscriptionNone,
	}
	storage.Instance().InsertOrUpdateRosterItem(ri)

	// insert roster approval notification...
	rn := &model.RosterNotification{
		Contact:  "noelia",
		JID:      "ortuman@jackal.im",
		Elements: []xml.XElement{},
	}
	storage.Instance().InsertOrUpdateRosterNotification(rn)

	stm1, stm2 := tUtilRosterInitializeRoster()

	r1 := New(&Config{}, stm1)
	r2 := New(&Config{}, stm2)

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
	require.Equal(t, SubscriptionFrom, iRes.Attributes().Get("subscription"))

	rns, err := storage.Instance().FetchRosterNotifications("noelia")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))

	ri, err = storage.Instance().FetchRosterItem("ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Equal(t, SubscriptionTo, ri.Subscription)
}

func TestRoster_Unsubscribe(t *testing.T) {
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	tUtilRosterInsertRosterItems()
	stm1, stm2 := tUtilRosterInitializeRoster()

	r1 := New(&Config{}, stm1)
	r2 := New(&Config{}, stm2)

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
	require.Equal(t, SubscriptionTo, iRes.Attributes().Get("subscription"))

	elem = stm2.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.UnsubscribeType, elem.Type())
}

func TestRoster_Unsubscribed(t *testing.T) {
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	tUtilRosterInsertRosterItems()
	stm1, stm2 := tUtilRosterInitializeRoster()

	r1 := New(&Config{}, stm1)
	r2 := New(&Config{}, stm2)

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
	require.Equal(t, SubscriptionFrom, iRes.Attributes().Get("subscription"))

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
	router.Initialize(&router.Config{Domains: []string{"jackal.im"}}, nil)
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
	}()

	tUtilRosterInsertRosterItems()
	stm1, stm2 := tUtilRosterInitializeRoster()

	r1 := New(&Config{}, stm1)
	r2 := New(&Config{}, stm2)

	tUtilRosterRequestRoster(r1, stm1)
	tUtilRosterRequestRoster(r2, stm2)

	// delete item IQ...
	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	item := xml.NewElementName("item")
	item.SetAttribute("jid", "noelia@jackal.im")
	item.SetAttribute("subscription", SubscriptionRemove)
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
	require.Equal(t, SubscriptionRemove, iRes.Attributes().Get("subscription"))

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
	require.Equal(t, SubscriptionTo, iRes.Attributes().Get("subscription"))

	elem = stm2.FetchElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, xml.SetType, elem.Type())
	qRes = elem.Elements().ChildNamespace("query", rosterNamespace)
	require.NotNil(t, qRes)
	iRes = qRes.Elements().Child("item")
	require.Equal(t, SubscriptionNone, iRes.Attributes().Get("subscription"))

	elem = stm2.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, xml.UnsubscribeType, elem.Type())
	require.Equal(t, "ortuman@jackal.im", elem.From())
}

func tUtilRosterInsertRosterItems() {
	// insert roster item...
	ri1 := &model.RosterItem{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Name:         "My Juliet",
		Subscription: SubscriptionBoth,
	}
	ri2 := &model.RosterItem{
		Username:     "noelia",
		JID:          "ortuman@jackal.im",
		Name:         "My Romeo",
		Subscription: SubscriptionBoth,
	}
	storage.Instance().InsertOrUpdateRosterItem(ri1)
	storage.Instance().InsertOrUpdateRosterItem(ri2)
}

func tUtilRosterRequestRoster(r *Roster, stm *stream.MockC2S) {
	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.AppendElement(xml.NewElementNamespace("query", rosterNamespace))

	r.ProcessIQ(iq)
	_ = stm.FetchElement()
}

func tUtilRosterInitializeRoster() (*stream.MockC2S, *stream.MockC2S) {
	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	j2, _ := xml.NewJID("noelia", "jackal.im", "garden", true)

	stm1 := stream.NewMockC2S("abcd1234", j1)
	stm1.SetUsername("ortuman")
	stm1.SetDomain("jackal.im")
	stm1.SetResource("balcony")
	stm1.SetAuthenticated(true)
	stm1.Context().SetBool(true, rosterRequestedCtxKey)
	stm1.SetJID(j1)

	stm2 := stream.NewMockC2S("abcd5678", j2)
	stm2.SetUsername("noelia")
	stm2.SetDomain("jackal.im")
	stm2.SetResource("garden")
	stm2.SetAuthenticated(true)
	stm2.Context().SetBool(true, rosterRequestedCtxKey)
	stm2.SetJID(j2)

	// register streams...
	router.Instance().RegisterC2S(stm1)
	router.Instance().RegisterC2S(stm2)
	router.Instance().RegisterC2SResource(stm1)
	router.Instance().RegisterC2SResource(stm2)

	return stm1, stm2
}
