/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package presencehub

import (
	"context"
	"crypto/tls"
	"testing"

	c2srouter "github.com/ortuman/jackal/c2s/router"

	"github.com/ortuman/jackal/model"
	"github.com/ortuman/jackal/router"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestPresenceHub_RegisterPresence(t *testing.T) {
	r, s := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("noelia", "jackal.im", "balcony", true)
	j3, _ := jid.New("noelia", "jackal.im", "yard", true)

	p1 := xmpp.NewPresence(j1, j1, xmpp.AvailableType)
	p2 := xmpp.NewPresence(j2, j2, xmpp.AvailableType)
	p3 := xmpp.NewPresence(j3, j3, xmpp.AvailableType)

	_ = s.UpsertCapabilities(context.Background(), &model.Capabilities{
		Node:     "http://code.google.com/p/exodus",
		Ver:      "QgayPKawpkPSDYmwT/WM94uAlu0=",
		Features: []string{"princely_musings+notify"},
	})

	// register presence
	c := xmpp.NewElementNamespace("c", "http://jabber.org/protocol/caps")
	c.SetAttribute("hash", "sha-1")
	c.SetAttribute("node", "http://code.google.com/p/exodus")
	c.SetAttribute("ver", "QgayPKawpkPSDYmwT/WM94uAlu0=")
	p2.AppendElement(c)

	ph := New(r, s)
	_, _ = ph.RegisterPresence(context.Background(), p1)
	_, _ = ph.RegisterPresence(context.Background(), p2)
	_, _ = ph.RegisterPresence(context.Background(), p3)

	availablePresences := ph.AvailablePresencesMatchingJID(j3.ToBareJID())
	require.Len(t, availablePresences, 2)

	ph.UnregisterPresence(context.Background(), p3)

	availablePresences = ph.AvailablePresencesMatchingJID(j2.ToBareJID())
	require.Len(t, availablePresences, 1)

	// check capabilities
	caps := availablePresences[0].Caps
	require.NotNil(t, caps)
	require.Equal(t, "http://code.google.com/p/exodus", caps.Node)
	require.Equal(t, "QgayPKawpkPSDYmwT/WM94uAlu0=", caps.Ver)
}

func TestPresenceHub_RequestCapabilities(t *testing.T) {
	r, s := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(context.Background(), stm1)

	// register presence
	p := xmpp.NewPresence(j1, j1, xmpp.AvailableType)
	c := xmpp.NewElementNamespace("c", "http://jabber.org/protocol/caps")
	c.SetAttribute("hash", "sha-1")
	c.SetAttribute("node", "http://code.google.com/p/exodus")
	c.SetAttribute("ver", "QgayPKawpkPSDYmwT/WM94uAlu0=")
	p.AppendElement(c)

	ph := New(r, s)
	_, _ = ph.RegisterPresence(context.Background(), p)

	elem := stm1.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, "jackal.im", elem.From())

	queryElem := elem.Elements().Child("query")
	require.NotNil(t, queryElem)

	require.Equal(t, "http://jabber.org/protocol/disco#info", queryElem.Namespace())
	require.Equal(t, "http://code.google.com/p/exodus#QgayPKawpkPSDYmwT/WM94uAlu0=", queryElem.Attributes().Get("node"))
}

func TestPresenceHub_ProcessCapabilities(t *testing.T) {
	r, s := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	iqID := uuid.New()

	iqRes := xmpp.NewIQType(iqID, xmpp.ResultType)
	iqRes.SetFromJID(j1)
	iqRes.SetToJID(j1.ToBareJID())

	qElem := xmpp.NewElementNamespace("query", "http://jabber.org/protocol/disco#info")
	qElem.SetAttribute("node", "http://code.google.com/p/exodus#QgayPKawpkPSDYmwT/WM94uAlu0=")
	featureEl := xmpp.NewElementName("feature")
	featureEl.SetAttribute("var", "cool+feature")
	qElem.AppendElement(featureEl)
	iqRes.AppendElement(qElem)

	ph := New(r, s)
	ph.activeDiscoInfo.Store(iqID, true)

	ph.processIQ(context.Background(), iqRes)

	// check storage capabilities
	caps, _ := s.FetchCapabilities(context.Background(), "http://code.google.com/p/exodus", "QgayPKawpkPSDYmwT/WM94uAlu0=")
	require.NotNil(t, caps)

	require.Len(t, caps.Features, 1)
	require.Equal(t, "cool+feature", caps.Features[0])
}

func setupTest(domain string) (router.GlobalRouter, *memorystorage.Capabilities) {
	s := memorystorage.NewCapabilities()
	r, _ := router.New(
		&router.Config{
			Hosts: []router.HostConfig{{Name: domain, Certificate: tls.Certificate{}}},
		},
		c2srouter.New(memorystorage.NewUser()),
		memorystorage.NewBlockList(),
	)
	return r, s
}
