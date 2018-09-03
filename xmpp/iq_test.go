/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp_test

import (
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/ortuman/jackal/xmpp/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestIQBuild(t *testing.T) {
	j, _ := jid.New("ortuman", "example.org", "balcony", false)

	elem := xmpp.NewElementName("message")
	_, err := xmpp.NewIQFromElement(elem, j, j) // wrong name...
	require.NotNil(t, err)

	elem.SetName("iq")
	_, err = xmpp.NewIQFromElement(elem, j, j) // no ID...
	require.NotNil(t, err)

	elem.SetID(uuid.New())
	_, err = xmpp.NewIQFromElement(elem, j, j) // no type...
	require.NotNil(t, err)

	elem.SetType("invalid")
	_, err = xmpp.NewIQFromElement(elem, j, j) // invalid type...
	require.NotNil(t, err)

	elem.SetType(xmpp.GetType)
	_, err = xmpp.NewIQFromElement(elem, j, j) // 'get' with no child...
	require.NotNil(t, err)

	elem.SetType(xmpp.ResultType)
	elem.AppendElements([]xmpp.XElement{xmpp.NewElementName("a"), xmpp.NewElementName("b")})
	_, err = xmpp.NewIQFromElement(elem, j, j) // 'result' with more than one child...
	require.NotNil(t, err)

	elem.SetType(xmpp.ResultType)
	elem.ClearElements()
	elem.AppendElements([]xmpp.XElement{xmpp.NewElementName("a")})
	iq, err := xmpp.NewIQFromElement(elem, j, j) // valid IQ...
	require.Nil(t, err)
	require.NotNil(t, iq)
}

func TestIQType(t *testing.T) {
	require.True(t, xmpp.NewIQType(uuid.New(), xmpp.GetType).IsGet())
	require.True(t, xmpp.NewIQType(uuid.New(), xmpp.SetType).IsSet())
	require.True(t, xmpp.NewIQType(uuid.New(), xmpp.ResultType).IsResult())
}

func TestResultIQ(t *testing.T) {
	j, _ := jid.New("", "jackal.im", "", true)

	id := uuid.New()
	iq := xmpp.NewIQType(id, xmpp.GetType)
	iq.SetFromJID(j)
	iq.SetToJID(j)
	iq.AppendElement(xmpp.NewElementNamespace("ping", "urn:xmpp:ping"))
	result := iq.ResultIQ()
	require.Equal(t, xmpp.ResultType, result.Type())
	require.Equal(t, id, result.ID())
}

func TestIQJID(t *testing.T) {
	from, _ := jid.New("ortuman", "test.org", "balcony", false)
	to, _ := jid.New("ortuman", "example.org", "garden", false)
	iq := xmpp.NewIQType(uuid.New(), xmpp.GetType)
	iq.SetFromJID(from)
	require.Equal(t, iq.FromJID().String(), iq.From())
	iq.SetToJID(to)
	require.Equal(t, iq.ToJID().String(), iq.To())
}
