/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0199

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/ortuman/jackal/router/host"

	c2srouter "github.com/ortuman/jackal/c2s/router"
	"github.com/ortuman/jackal/router"
	memorystorage "github.com/ortuman/jackal/storage/memory"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0199_Matching(t *testing.T) {
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	x := New(&Config{}, nil, nil)
	defer func() { _ = x.Shutdown() }()

	// test MatchesIQ
	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(j)

	ping := xmpp.NewElementNamespace("ping", pingNamespace)
	iq.AppendElement(ping)

	require.True(t, x.MatchesIQ(iq))
}

func TestXEP0199_ReceivePing(t *testing.T) {
	r := setupTest()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("juliet", "jackal.im", "garden", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(context.Background(), stm)

	x := New(&Config{}, nil, r)
	defer func() { _ = x.Shutdown() }()

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j2)

	x.ProcessIQ(context.Background(), iq)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetToJID(j1)
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	ping := xmpp.NewElementNamespace("ping", pingNamespace)
	iq.AppendElement(ping)

	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xmpp.GetType)
	x.ProcessIQ(context.Background(), iq)
	elem = stm.ReceiveElement()
	require.Equal(t, iqID, elem.ID())
}

func TestXEP0199_SendPing(t *testing.T) {
	r := setupTest()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("", "jackal.im", "", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(context.Background(), stm)

	x := New(&Config{Send: true, SendInterval: time.Second}, nil, r)
	defer func() { _ = x.Shutdown() }()

	x.SchedulePing(stm)

	// wait for ping...
	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("ping", pingNamespace))

	// send pong...
	pong := xmpp.NewIQType(elem.ID(), xmpp.ResultType)
	pong.SetFromJID(j1)
	pong.SetToJID(j2)
	x.ProcessIQ(context.Background(), pong)
	x.SchedulePing(stm)

	// wait next ping...
	elem = stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("ping", pingNamespace))

	// expect disconnection...
	err := stm.WaitDisconnection()
	require.NotNil(t, err)
}

func TestXEP0199_Disconnect(t *testing.T) {
	r := setupTest()

	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	r.Bind(context.Background(), stm)

	x := New(&Config{Send: true, SendInterval: time.Second}, nil, r)
	defer func() { _ = x.Shutdown() }()

	x.SchedulePing(stm)

	// wait next ping...
	elem := stm.ReceiveElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("ping", pingNamespace))

	// expect disconnection...
	err := stm.WaitDisconnection()
	require.NotNil(t, err)
	require.Equal(t, "connection-timeout", err.Error())
}

func setupTest() router.Router {
	hosts, _ := host.New([]host.Config{{Name: "jackal.im", Certificate: tls.Certificate{}}})
	r, _ := router.New(
		hosts,
		c2srouter.New(memorystorage.NewUser(), memorystorage.NewBlockList()),
		nil,
	)
	return r
}
