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

func TestXEP0191_Matching(t *testing.T) {
	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	x := NewXEPBlockingCommand(nil)
	defer x.Done()

	require.Equal(t, []string{blockingCommandNamespace}, x.AssociatedNamespaces())

	// test MatchesIQ
	iq1 := xml.NewIQType(uuid.New(), xml.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j)
	iq1.AppendElement(xml.NewElementNamespace("blocklist", blockingCommandNamespace))
	require.True(t, x.MatchesIQ(iq1))

	iq2 := xml.NewIQType(uuid.New(), xml.SetType)
	iq2.SetFromJID(j)
	iq2.SetToJID(j)
	iq2.AppendElement(xml.NewElementNamespace("block", blockingCommandNamespace))
	require.True(t, x.MatchesIQ(iq2))

	iq3 := xml.NewIQType(uuid.New(), xml.SetType)
	iq3.SetFromJID(j)
	iq3.SetToJID(j)
	iq3.AppendElement(xml.NewElementNamespace("unblock", blockingCommandNamespace))
	require.True(t, x.MatchesIQ(iq2))
}

func TestXEP0191_GetBlockList(t *testing.T) {
	storage.Initialize(&config.Storage{Type: config.Mock})
	defer storage.Shutdown()

	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	stm := c2s.NewMockStream(uuid.New(), j)

	x := NewXEPBlockingCommand(stm)
	defer x.Done()

	storage.Instance().InsertOrUpdateBlockListItems([]model.BlockListItem{{
		Username: "ortuman",
		JID:      "hamlet@jackal.im/garden",
	}, {
		Username: "ortuman",
		JID:      "jabber.org",
	}})

	iq1 := xml.NewIQType(uuid.New(), xml.GetType)
	iq1.SetFromJID(j)
	iq1.SetToJID(j)
	iq1.AppendElement(xml.NewElementNamespace("blocklist", blockingCommandNamespace))

	x.ProcessIQ(iq1)
	elem := stm.FetchElement()
	bl := elem.Elements().ChildNamespace("blocklist", blockingCommandNamespace)
	require.NotNil(t, bl)
	require.Equal(t, 2, len(bl.Elements().Children("item")))

	storage.ActivateMockedError()
	x.ProcessIQ(iq1)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	storage.DeactivateMockedError()
}
