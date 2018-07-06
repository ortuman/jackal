/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package roster

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

func TestPresenceHandler_Available(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	router.Initialize(&router.Config{})
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
		host.Shutdown()
	}()
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "garden", true)
	j3, _ := jid.New("fruela", "jackal.im", "balcony", true)
	j4, _ := jid.New("ortuman", "jackal.im", "yard", true)
	j5, _ := jid.New("boss", "jabber.org", "yard", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm1.SetAuthenticated(true)
	stm2 := stream.NewMockC2S(uuid.New(), j2)
	stm2.SetAuthenticated(true)

	router.Bind(stm1)
	router.Bind(stm2)

	// user entity
	storage.Instance().InsertOrUpdateUser(&model.User{
		Username:     "ortuman",
		LastPresence: xml.NewPresence(j1, j1.ToBareJID(), xml.UnavailableType),
	})

	// roster items
	storage.Instance().InsertOrUpdateRosterItem(&rostermodel.Item{
		Username:     "noelia",
		JID:          "ortuman@jackal.im",
		Subscription: rostermodel.SubscriptionBoth,
	})
	storage.Instance().InsertOrUpdateRosterItem(&rostermodel.Item{
		Username:     "ortuman",
		JID:          "noelia@jackal.im",
		Subscription: rostermodel.SubscriptionBoth,
	})

	// pending notification
	storage.Instance().InsertOrUpdateRosterNotification(&rostermodel.Notification{
		Contact:  "ortuman",
		JID:      j3.ToBareJID().String(),
		Presence: xml.NewPresence(j3.ToBareJID(), j1.ToBareJID(), xml.SubscribeType),
	})

	ph := NewPresenceHandler(&Config{})

	// online presence...
	ph.ProcessPresence(xml.NewPresence(j1, j1.ToBareJID(), xml.AvailableType))

	// receive pending approval notification...
	elem := stm1.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, j3.ToBareJID().String(), elem.From())
	require.Equal(t, xml.SubscribeType, elem.Type())

	// expect user's available presence
	elem = stm2.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, j1.String(), elem.From())
	require.Equal(t, xml.AvailableType, elem.Type())

	// check if last presence was updated
	usr, err := storage.Instance().FetchUser("ortuman")
	require.Nil(t, err)
	require.NotNil(t, usr)
	require.NotNil(t, usr.LastPresence)
	require.Equal(t, xml.AvailableType, usr.LastPresence.Type())

	// send remaining online presences...
	ph.ProcessPresence(xml.NewPresence(j2, j2.ToBareJID(), xml.AvailableType))
	ph.ProcessPresence(xml.NewPresence(j3, j3.ToBareJID(), xml.AvailableType))
	ph.ProcessPresence(xml.NewPresence(j4, j1.ToBareJID(), xml.AvailableType))
	ph.ProcessPresence(xml.NewPresence(j5, j1.ToBareJID(), xml.AvailableType))

	require.Equal(t, 1, len(OnlinePresencesMatchingJID(j1)))

	j6, _ := jid.NewWithString("jackal.im", true)
	require.Equal(t, 4, len(OnlinePresencesMatchingJID(j6)))

	j7, _ := jid.NewWithString("jabber.org", true)
	require.Equal(t, 1, len(OnlinePresencesMatchingJID(j7)))

	j8, _ := jid.NewWithString("jackal.im/balcony", true)
	require.Equal(t, 2, len(OnlinePresencesMatchingJID(j8)))

	j9, _ := jid.NewWithString("ortuman@jackal.im", true)
	require.Equal(t, 2, len(OnlinePresencesMatchingJID(j9)))

	// send unavailable presences...
	ph.ProcessPresence(xml.NewPresence(j1, j1.ToBareJID(), xml.UnavailableType))
	ph.ProcessPresence(xml.NewPresence(j2, j2.ToBareJID(), xml.UnavailableType))
	ph.ProcessPresence(xml.NewPresence(j3, j3.ToBareJID(), xml.UnavailableType))
	ph.ProcessPresence(xml.NewPresence(j4, j4.ToBareJID(), xml.UnavailableType))
	ph.ProcessPresence(xml.NewPresence(j5, j1.ToBareJID(), xml.UnavailableType))

	require.Equal(t, 0, len(OnlinePresencesMatchingJID(j1)))
	require.Equal(t, 0, len(OnlinePresencesMatchingJID(j6)))
	require.Equal(t, 0, len(OnlinePresencesMatchingJID(j7)))
	require.Equal(t, 0, len(OnlinePresencesMatchingJID(j8)))
	require.Equal(t, 0, len(OnlinePresencesMatchingJID(j9)))
}

func TestPresenceHandler_Probe(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	router.Initialize(&router.Config{})
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
		host.Shutdown()
	}()
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "garden", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	stm.SetAuthenticated(true)

	router.Bind(stm)

	ph := NewPresenceHandler(&Config{})

	// user doesn't exist...
	ph.ProcessPresence(xml.NewPresence(j1, j2, xml.ProbeType))
	elem := stm.FetchElement()
	require.Equal(t, "presence", elem.Name())
	require.Equal(t, "noelia@jackal.im", elem.From())
	require.Equal(t, xml.UnsubscribedType, elem.Type())

	storage.Instance().InsertOrUpdateUser(&model.User{
		Username:     "noelia",
		LastPresence: xml.NewPresence(j2.ToBareJID(), j2.ToBareJID(), xml.UnavailableType),
	})

	// user exists, with no presence subscription...
	ph.ProcessPresence(xml.NewPresence(j1, j2, xml.ProbeType))
	elem = stm.FetchElement()
	require.Equal(t, xml.UnsubscribedType, elem.Type())

	storage.Instance().InsertOrUpdateRosterItem(&rostermodel.Item{
		Username:     "noelia",
		JID:          "ortuman@jackal.im",
		Subscription: rostermodel.SubscriptionFrom,
	})
	ph.ProcessPresence(xml.NewPresence(j1, j2, xml.ProbeType))
	elem = stm.FetchElement()
	require.Equal(t, xml.UnavailableType, elem.Type())

	// test available presence...
	p2 := xml.NewPresence(j2, j2.ToBareJID(), xml.AvailableType)
	storage.Instance().InsertOrUpdateUser(&model.User{
		Username:     "noelia",
		LastPresence: p2,
	})
	ph.ProcessPresence(xml.NewPresence(j1, j2, xml.ProbeType))
	elem = stm.FetchElement()
	require.Equal(t, xml.AvailableType, elem.Type())
	require.Equal(t, "noelia@jackal.im/garden", elem.From())
}

func TestPresenceHandler_Subscription(t *testing.T) {
	host.Initialize([]host.Config{{Name: "jackal.im"}})
	router.Initialize(&router.Config{})
	storage.Initialize(&storage.Config{Type: storage.Memory})
	defer func() {
		router.Shutdown()
		storage.Shutdown()
		host.Shutdown()
	}()
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "garden", true)

	ph := NewPresenceHandler(&Config{})
	ph.ProcessPresence(xml.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xml.SubscribeType))

	rns, err := storage.Instance().FetchRosterNotifications("noelia")
	require.Nil(t, err)
	require.Equal(t, 1, len(rns))

	// resend request...
	require.Nil(t, ph.ProcessPresence(xml.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xml.SubscribeType)))

	// contact request cancellation
	ph.ProcessPresence(xml.NewPresence(j2.ToBareJID(), j1.ToBareJID(), xml.UnsubscribedType))
	rns, err = storage.Instance().FetchRosterNotifications("noelia")
	require.Nil(t, err)
	require.Equal(t, 0, len(rns))

	ri, err := storage.Instance().FetchRosterItem("ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionNone, ri.Subscription)

	// contact accepts request...
	ph.ProcessPresence(xml.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xml.SubscribeType))
	ph.ProcessPresence(xml.NewPresence(j2.ToBareJID(), j1.ToBareJID(), xml.SubscribedType))

	ri, err = storage.Instance().FetchRosterItem("ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionTo, ri.Subscription)

	// contact subscribes to user's presence...
	ph.ProcessPresence(xml.NewPresence(j2.ToBareJID(), j1.ToBareJID(), xml.SubscribeType))
	ph.ProcessPresence(xml.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xml.SubscribedType))

	ri, err = storage.Instance().FetchRosterItem("noelia", "ortuman@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionBoth, ri.Subscription)

	// user unsubscribes from contact's presence...
	ph.ProcessPresence(xml.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xml.UnsubscribeType))

	ri, err = storage.Instance().FetchRosterItem("ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionFrom, ri.Subscription)

	// user cancels contact subscription
	ph.ProcessPresence(xml.NewPresence(j1.ToBareJID(), j2.ToBareJID(), xml.UnsubscribedType))
	ri, err = storage.Instance().FetchRosterItem("ortuman", "noelia@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionNone, ri.Subscription)

	ri, err = storage.Instance().FetchRosterItem("noelia", "ortuman@jackal.im")
	require.Nil(t, err)
	require.Equal(t, rostermodel.SubscriptionNone, ri.Subscription)
}
