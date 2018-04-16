/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package module

import (
	"testing"

	"time"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/model"
	"github.com/ortuman/jackal/stream/c2s"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0012_Matching(t *testing.T) {
	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	x := NewXEPLastActivity(nil)
	defer x.Done()

	require.Equal(t, []string{lastActivityNamespace}, x.AssociatedNamespaces())

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
	j1, _ := xml.NewJID("", "jackal.im", "", true)
	j2, _ := xml.NewJID("ortuman", "jackal.im", "garden", true)
	stm := c2s.NewMockStream("abcd", j2)

	x := NewXEPLastActivity(stm)
	defer x.Done()

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
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	c2s.Initialize(&config.C2S{})
	defer c2s.Shutdown()

	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	j2, _ := xml.NewJID("noelia", "jackal.im", "", true)
	stm1 := c2s.NewMockStream("abcd", j1)
	stm2 := c2s.NewMockStream("abcde", j2)
	stm2.SetResource("a_res")

	x := NewXEPLastActivity(stm1)
	defer x.Done()

	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j2)
	iq.AppendElement(xml.NewElementNamespace("query", lastActivityNamespace))

	x.ProcessIQ(iq)
	elem := stm1.FetchElement()
	require.Equal(t, xml.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	storage.Instance().InsertOrUpdateUser(&model.User{
		Username:        "noelia",
		LoggedOutStatus: "Gone!",
		LoggedOutAt:     time.Now().AddDate(0, 0, -1),
	})
	storage.Instance().InsertOrUpdateRosterItem(&model.RosterItem{
		User:         "ortuman",
		Contact:      "noelia",
		Subscription: subscriptionBoth,
	})
	x.ProcessIQ(iq)
	elem = stm1.FetchElement()
	q := elem.Elements().ChildNamespace("query", lastActivityNamespace)
	secs := q.Attributes().Get("seconds")
	require.True(t, len(secs) > 0)

	// set as online
	c2s.Instance().RegisterStream(stm2)
	c2s.Instance().AuthenticateStream(stm2)

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
