/*
 * Copyright (c) 2019 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0115

/*
func TestEntityCaps_RegisterPresence(t *testing.T) {
}

func TestEntityCaps_RequestCapabilities(t *testing.T) {
	r, s := setupTest("jackal.im")

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm1 := stream.NewMockC2S(uuid.New(), j1)
	stm1.SetPresence(xmpp.NewPresence(j1, j1, xmpp.AvailableType))

	r.Bind(context.Background(), stm1)

	// register presence
	p := xmpp.NewPresence(j1, j1, xmpp.AvailableType)
	c := xmpp.NewElementNamespace("c", "http://jabber.org/protocol/caps")
	c.SetAttribute("hash", "sha-1")
	c.SetAttribute("node", "http://code.google.com/p/exodus")
	c.SetAttribute("ver", "QgayPKawpkPSDYmwT/WM94uAlu0=")
	p.AppendElement(c)

	ph := New(r, s, "alloc-1234")
	_, _ = ph.RegisterPresence(context.Background(), p)

	elem := stm1.ReceiveElement()
	require.Equal(t, "iq", elem.Name())
	require.Equal(t, "jackal.im", elem.From())

	queryElem := elem.Elements().Child("query")
	require.NotNil(t, queryElem)

	require.Equal(t, "http://jabber.org/protocol/disco#info", queryElem.Namespace())
	require.Equal(t, "http://code.google.com/p/exodus#QgayPKawpkPSDYmwT/WM94uAlu0=", queryElem.Attributes().Get("node"))
}

func TestEntityCaps_ProcessCapabilities(t *testing.T) {
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

	ph := New(r, s, "alloc-1234")
	ph.activeDiscoInfo.Store(iqID, true)

	ph.processIQ(context.Background(), iqRes)

	// check storage capabilities
	caps, _ := s.FetchCapabilities(context.Background(), "http://code.google.com/p/exodus", "QgayPKawpkPSDYmwT/WM94uAlu0=")
	require.NotNil(t, caps)

	require.Len(t, caps.Features, 1)
	require.Equal(t, "cool+feature", caps.Features[0])
}

func setupTest(domain string) (router.Router, *memorystorage.Presences) {
	hosts, _ := host.New([]host.Config{{Name: domain, Certificate: tls.Certificate{}}})

	s := memorystorage.NewPresences()
	r, _ := router.New(
		hosts,
		c2srouter.New(memorystorage.NewUser(), memorystorage.NewBlockList()),
		nil,
	)
	return r, s
}
*/
