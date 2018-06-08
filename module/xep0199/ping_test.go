/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0199

import (
	"testing"

	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0199_Matching(t *testing.T) {
	t.Parallel()
	j, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	x := New(&Config{}, nil)

	// test MatchesIQ
	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.GetType)
	iq.SetFromJID(j)

	ping := xml.NewElementNamespace("ping", pingNamespace)
	iq.AppendElement(ping)

	require.True(t, x.MatchesIQ(iq))
}

func TestXEP0199_ReceivePing(t *testing.T) {
	t.Parallel()
	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)
	j2, _ := xml.NewJID("juliet", "jackal.im", "garden", true)

	stm := stream.NewMockC2S("abcd", j1)
	defer stm.Disconnect(nil)

	stm.SetUsername("ortuman")

	x := New(&Config{}, stm)

	iqID := uuid.New()
	iq := xml.NewIQType(iqID, xml.SetType)
	iq.SetFromJID(j2)
	iq.SetToJID(j2)

	x.ProcessIQ(iq)
	elem := stm.FetchElement()
	require.Equal(t, xml.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetToJID(j1)
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	ping := xml.NewElementNamespace("ping", pingNamespace)
	iq.AppendElement(ping)

	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, xml.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xml.GetType)
	x.ProcessIQ(iq)
	elem = stm.FetchElement()
	require.Equal(t, iqID, elem.ID())
}

func TestXEP0199_SendPing(t *testing.T) {
	t.Parallel()
	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd", j1)
	defer stm.Disconnect(nil)

	stm.SetUsername("ortuman")

	x := New(&Config{Send: true, SendInterval: 1}, stm)

	x.StartPinging()

	// wait for ping...
	elem := stm.FetchElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("ping", pingNamespace))

	// send pong...
	x.ProcessIQ(xml.NewIQType(elem.ID(), xml.ResultType))
	x.ResetDeadline()

	// wait next ping...
	elem = stm.FetchElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("ping", pingNamespace))

	// expect disconnection...
	err := stm.WaitDisconnection()
	require.NotNil(t, err)
}

func TestXEP0199_Disconnect(t *testing.T) {
	t.Parallel()
	j1, _ := xml.NewJID("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd", j1)
	defer stm.Disconnect(nil)

	stm.SetUsername("ortuman")

	x := New(&Config{Send: true, SendInterval: 1}, stm)

	x.StartPinging()

	// wait next ping...
	elem := stm.FetchElement()
	require.NotNil(t, elem)
	require.Equal(t, "iq", elem.Name())
	require.NotNil(t, elem.Elements().ChildNamespace("ping", pingNamespace))

	// expect disconnection...
	err := stm.WaitDisconnection()
	require.NotNil(t, err)
	require.Equal(t, "connection-timeout", err.Error())
}
