/*
 * Copyright (c) 2020 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package c2srouter

import (
	"context"
	"testing"

	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/stretchr/testify/require"
)

func TestResources_Binding(t *testing.T) {
	j, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	stm := stream.NewMockC2S("id-1", j)

	res := resources{}
	require.Equal(t, 0, res.len())

	res.bind(stm)
	require.Equal(t, 1, res.len())

	require.NotNil(t, res.stream("yard"))
	require.Len(t, res.allStreams(), 1)

	res.unbind("yard")

	require.Nil(t, res.stream("yard"))
	require.Len(t, res.allStreams(), 0)
}

func TestResources_Route(t *testing.T) {
	j1, _ := jid.NewWithString("ortuman@jackal.im/yard", true)
	j2, _ := jid.NewWithString("ortuman@jackal.im/balcony", true)
	j3, _ := jid.NewWithString("ortuman@jackal.im/chamber", true)

	stm1 := stream.NewMockC2S("id-1", j1)
	stm2 := stream.NewMockC2S("id-2", j2)

	res := resources{}
	res.bind(stm1)
	res.bind(stm2)

	err := res.route(context.Background(), xmpp.NewPresence(j1, j3, xmpp.AvailableType))
	require.Equal(t, router.ErrResourceNotFound, err)
}
