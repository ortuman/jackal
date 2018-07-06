/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

import (
	"testing"

	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestRoster_MatchesIQ(t *testing.T) {
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

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

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

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

	ri1 := &rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Name:         "My Juliet",
		Subscription: rostermodel.SubscriptionNone,
		Ask:          true,
		Groups:       []string{"people", "friends"},
	}
	storage.Instance().InsertOrUpdateRosterItem(ri1)

	ri2 := &rostermodel.Item{
		Username:     "ortuman",
		JID:          "romeo@jackal.im",
		Name:         "Rome",
		Subscription: rostermodel.SubscriptionNone,
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

func TestRoster_Update(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	router.Initialize(&router.Config{})
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
		host.Shutdown()
	}()

	j1, _ := jid.New("ortuman", "jackal.im", "garden", true)
	j2, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm1.SetAuthenticated(true)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	stm2.SetAuthenticated(true)
	stm2.Context().SetBool(true, rosterRequestedCtxKey)

	r := New(&Config{}, stm1)

	router.Bind(stm1)
	router.Bind(stm2)

	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	q := xml.NewElementNamespace("query", rosterNamespace)
	item := xml.NewElementName("item")
	item.SetAttribute("jid", "noelia@jackal.im")
	item.SetAttribute("subscription", rostermodel.SubscriptionNone)
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

	// expecting roster push...
	elem = stm2.FetchElement()
	require.Equal(t, xml.SetType, elem.Type())

	// update name
	item.SetAttribute("name", "My Girl")
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
	require.Equal(t, "My Girl", ri.Name)
}

func TestRoster_RemoveItem(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	router.Initialize(&router.Config{})
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
		host.Shutdown()
	}()

	// insert contact's roster item
	storage.Instance().InsertOrUpdateRosterItem(&rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Name:         "My Juliet",
		Subscription: rostermodel.SubscriptionBoth,
	})
	storage.Instance().InsertOrUpdateRosterItem(&rostermodel.Item{
		Username:     "noelia",
		JID:          "ortuman@jackal.im",
		Name:         "My Romeo",
		Subscription: rostermodel.SubscriptionBoth,
	})
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd", j1)
	stm.SetUsername("ortuman")
	stm.SetDomain("jackal.im")

	r := New(&Config{}, stm)
	defer stm.Disconnect(nil)

	// remove item
	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	q := xml.NewElementNamespace("query", rosterNamespace)
	item := xml.NewElementName("item")
	item.SetAttribute("jid", "noelia@jackal.im")
	item.SetAttribute("subscription", rostermodel.SubscriptionRemove)
	q.AppendElement(item)
	iq.AppendElement(q)

	r.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, iqID, elem.ID())

	ri, err := storage.Instance().FetchRosterItem("ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Nil(t, ri)
}
