/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0012

import (
	"testing"

	"github.com/ortuman/jackal/host"
	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/model/rostermodel"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0012_Matching(t *testing.T) {
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	x := New(nil)

	// test MatchesIQ
	iq1 := xml.NewIQType(uuid.New(), xml.GetType)
	iq1.SetFromJID(j)

	require.False(t, x.MatchesIQ(iq1))

	iq1.AppendElement(xml.NewElementNamespace("query", lastActivityNamespace))

	iq2 := xml.NewIQType(uuid.New(), xml.GetType)
	iq2.SetFromJID(j)
	iq2.AppendElement(xml.NewElementNamespace("query", lastActivityNamespace))

	require.True(t, x.MatchesIQ(iq1))
	require.True(t, x.MatchesIQ(iq2))

	iq1.SetType(xml.SetType)
	iq2.SetType(xml.ResultType)

	require.False(t, x.MatchesIQ(iq1))
	require.False(t, x.MatchesIQ(iq2))
}

func TestXEP0012_GetServerLastActivity(t *testing.T) {
	j1, _ := jid.New("", "jackal.im", "", true)
	j2, _ := jid.New("ortuman", "jackal.im", "garden", true)

	stm := stream.NewMockC2S("abcd", j2)
	defer stm.Disconnect(nil)

	x := New(stm)

	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.SetToJID(j1)
	iq.AppendElement(xml.NewElementNamespace("query", lastActivityNamespace))

	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	q := elem.Elements().Child("query")
	require.NotNil(t, q)
	secs := q.Attributes().Get("seconds")
	require.True(t, len(secs) > 0)
}

func TestXEP0012_GetOnlineUserLastActivity(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	storage.Initialize(&storage.Config{Type: storage.Memory})
	router.Initialize(&router.Config{})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
		host.Shutdown()
	}()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "", true)
	stm1 := stream.NewMockC2S("abcd", j1)
	stm2 := stream.NewMockC2S("abcde", j2)
	stm2.SetResource("a_res")

	x := New(stm1)

	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j2)
	iq.AppendElement(xml.NewElementNamespace("query", lastActivityNamespace))

	x.ProcessIQ(iq)
	elem := stm1.FetchElement()
	require.Equal(t, xml.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	p := xml.NewPresence(j1, j1, xml.UnavailableType)
	st := xml.NewElementName("status")
	st.SetText("Gone!")
	p.AppendElement(st)

	storage.Instance().InsertOrUpdateUser(&model.User{
		Username:     "noelia",
		LastPresence: p,
	})
	storage.Instance().InsertOrUpdateRosterItem(&rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: "both",
	})
	x.ProcessIQ(iq)
	elem = stm1.FetchElement()
	q := elem.Elements().ChildNamespace("query", lastActivityNamespace)
	secs := q.Attributes().Get("seconds")
	require.True(t, len(secs) > 0)

	// set as online
	router.Bind(stm2)

	x.ProcessIQ(iq)
	elem = stm1.FetchElement()
	q = elem.Elements().ChildNamespace("query", lastActivityNamespace)
	secs = q.Attributes().Get("seconds")
	require.Equal(t, "0", secs)

	storage.ActivateMockedError()
	x.ProcessIQ(iq)
	elem = stm1.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	storage.DeactivateMockedError()
}
