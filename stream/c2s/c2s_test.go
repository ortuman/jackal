/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2s

import (
	"testing"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestC2SManager(t *testing.T) {
	Initialize(&config.C2S{Domains: []string{"jackal.im"}})
	defer Shutdown()

	require.Equal(t, "jackal.im", Instance().DefaultLocalDomain())
	require.True(t, Instance().IsLocalDomain("jackal.im"))
	require.False(t, Instance().IsLocalDomain("example.org"))

	j1, _ := xml.NewJIDString("ortuman@jackal.im/balcony", false)
	j2, _ := xml.NewJIDString("ortuman@jackal.im/garden", false)
	j3, _ := xml.NewJIDString("ortuman@example.org/balcony", false)
	strm1 := NewMockStream(uuid.New(), j1)
	strm2 := NewMockStream(uuid.New(), j2)
	strm3 := NewMockStream(uuid.New(), j3)

	err := Instance().RegisterStream(strm1)
	require.Nil(t, err)
	err = Instance().RegisterStream(strm1) // already registered...
	require.NotNil(t, err)
	err = Instance().RegisterStream(strm2)
	require.Nil(t, err)
	err = Instance().RegisterStream(strm3)
	require.NotNil(t, err)

	strm1.SetResource("")
	err = Instance().AuthenticateStream(strm1) // resource not assigned...
	require.NotNil(t, err)
	strm1.SetResource("balcony")
	err = Instance().AuthenticateStream(strm1)
	require.Nil(t, err)
	err = Instance().AuthenticateStream(strm2)
	require.Nil(t, err)

	strms := Instance().AvailableStreams("ortuman")
	require.Equal(t, 2, len(strms))
	require.Equal(t, "ortuman@jackal.im/balcony", strms[0].JID().String())
	require.Equal(t, "ortuman@jackal.im/garden", strms[1].JID().String())

	err = Instance().UnregisterStream(strm1)
	require.Nil(t, err)
	err = Instance().UnregisterStream(strm1)
	require.NotNil(t, err) // already unregistered...
	err = Instance().UnregisterStream(strm2)
	require.Nil(t, err)

	strms = Instance().AvailableStreams("ortuman")
	require.Equal(t, 0, len(strms))
}
