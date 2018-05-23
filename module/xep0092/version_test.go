/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0092

import (
	"testing"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/version"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0092(t *testing.T) {
	srvJID, _ := xml.NewJID("", "jackal.im", "", true)
	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	stm := router.NewMockC2S("abcd", j)

	cfg := Config{}
	x := New(&cfg, stm, nil)

	// test MatchesIQ
	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.SetFromJID(j)
	iq.SetToJID(j)

	qVer := xml.NewElementNamespace("query", versionNamespace)

	iq.AppendElement(xml.NewElementNamespace("query", "jabber:client"))
	require.False(t, x.MatchesIQ(iq))
	iq.ClearElements()
	iq.AppendElement(qVer)
	require.False(t, x.MatchesIQ(iq))
	iq.SetToJID(srvJID)
	require.True(t, x.MatchesIQ(iq))

	qVer.AppendElement(xml.NewElementName("version"))
	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	// get version
	qVer.ClearElements()
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	ver := elem.Elements().ChildNamespace("query", versionNamespace)
	require.Equal(t, "jackal", ver.Elements().Child("name").Text())
	require.Equal(t, version.ApplicationVersion.String(), ver.Elements().Child("version").Text())
	require.Nil(t, ver.Elements().Child("os"))

	// show OS
	cfg.ShowOS = true

	x = New(&cfg, stm, nil)
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	ver = elem.Elements().ChildNamespace("query", versionNamespace)
	require.Equal(t, osString, ver.Elements().Child("os").Text())
}
