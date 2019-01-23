/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0092

import (
	"crypto/tls"
	"testing"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/version"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0092(t *testing.T) {
	r, _ := router.New(&router.Config{
		Hosts: []router.HostConfig{{Name: "jackal.im", Certificate: tls.Certificate{}}},
	})

	srvJID, _ := jid.New("", "jackal.im", "", true)
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j)
	r.Bind(stm)

	cfg := Config{}
	x := New(&cfg, nil)
	defer x.Shutdown()

	// test MatchesIQ
	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j)
	iq.SetToJID(j)

	qVer := xmpp.NewElementNamespace("query", versionNamespace)

	iq.AppendElement(xmpp.NewElementNamespace("query", "jabber:client"))
	require.False(t, x.MatchesIQ(iq))
	iq.ClearElements()
	iq.AppendElement(qVer)
	require.False(t, x.MatchesIQ(iq))
	iq.SetToJID(srvJID)
	require.True(t, x.MatchesIQ(iq))

	qVer.AppendElement(xmpp.NewElementName("version"))
	x.ProcessIQ(iq, r)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	// get version
	qVer.ClearElements()
	x.ProcessIQ(iq, r)
	elem = stm.ReceiveElement()
	ver := elem.Elements().ChildNamespace("query", versionNamespace)
	require.Equal(t, "jackal", ver.Elements().Child("name").Text())
	require.Equal(t, version.ApplicationVersion.String(), ver.Elements().Child("version").Text())
	require.Nil(t, ver.Elements().Child("os"))

	// show OS
	cfg.ShowOS = true

	x = New(&cfg, nil)
	defer x.Shutdown()

	x.ProcessIQ(iq, r)
	elem = stm.ReceiveElement()
	ver = elem.Elements().ChildNamespace("query", versionNamespace)
	require.Equal(t, osString, ver.Elements().Child("os").Text())
}
