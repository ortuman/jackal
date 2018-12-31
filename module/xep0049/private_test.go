/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0049

import (
	"testing"

	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/storage/memstorage"
	"github.com/ortuman/jackal/stream"
	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestXEP0049_Matching(t *testing.T) {
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("romeo", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd", j1)
	defer stm.Disconnect(nil)

	x := New()
	defer x.Shutdown()

	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j2.ToBareJID())
	require.False(t, x.MatchesIQ(iq))

	iq.AppendElement(xmpp.NewElementNamespace("query", privateNamespace))
	require.True(t, x.MatchesIQ(iq))
}

func TestXEP0049_InvalidIQ(t *testing.T) {
	j1, _ := jid.New("ortuman", "jackal.im", "balcony", true)
	j2, _ := jid.New("romeo", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd", j1)
	defer stm.Disconnect(nil)

	x := New()
	defer x.Shutdown()

	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(j1)
	iq.SetToJID(j2.ToBareJID())
	q := xmpp.NewElementNamespace("query", privateNamespace)
	iq.AppendElement(q)

	x.ProcessIQ(iq, stm)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrForbidden.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xmpp.ResultType)
	iq.SetToJID(j1.ToBareJID())
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())

	iq.SetType(xmpp.GetType)
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrNotAcceptable.Error(), elem.Error().Elements().All()[0].Name())

	exodus := xmpp.NewElementNamespace("exodus", "exodus:ns")
	exodus.AppendElement(xmpp.NewElementName("exodus2"))
	q.AppendElement(exodus)
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrNotAcceptable.Error(), elem.Error().Elements().All()[0].Name())

	exodus.ClearElements()
	exodus.SetNamespace("jabber:client")
	iq.SetType(xmpp.SetType)
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrNotAcceptable.Error(), elem.Error().Elements().All()[0].Name())

	exodus.SetNamespace("")
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrBadRequest.Error(), elem.Error().Elements().All()[0].Name())
}

func TestXEP0049_SetAndGetPrivate(t *testing.T) {
	s, shutdown := setupTest("jackal.im")
	defer shutdown()

	j, _ := jid.New("ortuman", "jackal.im", "balcony", true)

	stm := stream.NewMockC2S("abcd", j)
	defer stm.Disconnect(nil)

	x := New()
	defer x.Shutdown()

	iqID := uuid.New()
	iq := xmpp.NewIQType(iqID, xmpp.SetType)
	iq.SetFromJID(j)
	iq.SetToJID(j.ToBareJID())
	q := xmpp.NewElementNamespace("query", privateNamespace)
	iq.AppendElement(q)

	exodus1 := xmpp.NewElementNamespace("exodus1", "exodus:ns")
	exodus2 := xmpp.NewElementNamespace("exodus2", "exodus:ns")
	q.AppendElement(exodus1)
	q.AppendElement(exodus2)

	// set error
	s.EnableMockedError()
	x.ProcessIQ(iq, stm)
	elem := stm.ReceiveElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	s.DisableMockedError()

	// set success
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	// get error
	q.RemoveElements("exodus2")
	iq.SetType(xmpp.GetType)

	s.EnableMockedError()
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ErrInternalServerError.Error(), elem.Error().Elements().All()[0].Name())
	s.DisableMockedError()

	// get success
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())

	q2 := elem.Elements().ChildNamespace("query", privateNamespace)
	require.Equal(t, 2, q2.Elements().Count())
	require.Equal(t, "exodus:ns", q2.Elements().All()[0].Namespace())

	// get non existing
	exodus1.SetNamespace("exodus:ns:2")
	x.ProcessIQ(iq, stm)
	elem = stm.ReceiveElement()
	require.Equal(t, xmpp.ResultType, elem.Type())
	require.Equal(t, iqID, elem.ID())
	q3 := elem.Elements().ChildNamespace("query", privateNamespace)
	require.Equal(t, 1, q3.Elements().Count())
	require.Equal(t, "exodus:ns:2", q3.Elements().All()[0].Namespace())
}

func setupTest(domain string) (*memstorage.Storage, func()) {
	s := memstorage.New()
	storage.Set(s)
	return s, func() {
		storage.Unset()
	}
}
