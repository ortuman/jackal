/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0199

import (
	"testing"
	"time"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0199_Matching(t *testing.T) {
	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	x, shutdownCh := New(&Config{}, nil)
	defer close(shutdownCh)

	// test MatchesIQ
	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.GetType)
	iq.SetFromJID(j)

	ping := xmpp.NewElementNamespace("ping", pingNamespace)
	iq.AppendElement(ping)

	require.True(t, x.MatchesIQ(iq))
}

func TestXEP0199_ReceivePing(t *testing.T) {
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("juliet", "jackal.im", "garden", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	defer stm.Disconnect(nil)

	x, shutdownCh := New(&Config{}, nil)
	defer close(shutdownCh)

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j2)

	x.ProcessIQ(iq, stm)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetToJID(j1)
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	ping := xmpp.NewElementNamespace("ping", pingNamespace)
	iq.AppendElement(ping)

	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xmpp.GetType)
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, iqID, elem.ID())
}

func TestXEP0199_SendPing(t *testing.T) {
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("", "jackal.im", "", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	defer stm.Disconnect(nil)

	x, shutdownCh := New(&Config{Send: true, SendInterval: time.Second}, nil)
	defer close(shutdownCh)

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
	x.ProcessIQ(pong, stm)
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
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S(uuid.New(), j1)
	defer stm.Disconnect(nil)

	x, shutdownCh := New(&Config{Send: true, SendInterval: time.Second}, nil)
	defer close(shutdownCh)

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
