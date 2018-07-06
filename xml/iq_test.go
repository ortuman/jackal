/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml_test

import (
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/ortuman/jackal/xml/jid"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestIQBuild(t *testing.T) {
	j, _ := jid.New("ortuman", "example.org", "balcony", false)

	elem := xml.NewElementName("message")
	_, err := xml.NewIQFromElement(elem, j, j) // wrong name...
	require.NotNil(t, err)

	elem.SetName("iq")
	_, err = xml.NewIQFromElement(elem, j, j) // no ID...
	require.NotNil(t, err)

	elem.SetID(uuid.New())
	_, err = xml.NewIQFromElement(elem, j, j) // no type...
	require.NotNil(t, err)

	elem.SetType("invalid")
	_, err = xml.NewIQFromElement(elem, j, j) // invalid type...
	require.NotNil(t, err)

	elem.SetType(xml.GetType)
	_, err = xml.NewIQFromElement(elem, j, j) // 'get' with no child...
	require.NotNil(t, err)

	elem.SetType(xml.ResultType)
	elem.AppendElements([]xml.XElement{xml.NewElementName("a"), xml.NewElementName("b")})
	_, err = xml.NewIQFromElement(elem, j, j) // 'result' with more than one child...
	require.NotNil(t, err)

	elem.SetType(xml.ResultType)
	elem.ClearElements()
	elem.AppendElements([]xml.XElement{xml.NewElementName("a")})
	iq, err := xml.NewIQFromElement(elem, j, j) // valid IQ...
	require.Nil(t, err)
	require.NotNil(t, iq)
}

func TestIQType(t *testing.T) {
	require.True(t, xml.NewIQType(uuid.New(), xml.GetType).IsGet())
	require.True(t, xml.NewIQType(uuid.New(), xml.SetType).IsSet())
	require.True(t, xml.NewIQType(uuid.New(), xml.ResultType).IsResult())
}

func TestResultIQ(t *testing.T) {
	id := uuid.New()
	iq := xml.NewIQType(id, xml.GetType)
	iq.AppendElement(xml.NewElementNamespace("ping", "urn:xmpp:ping"))
	result := iq.ResultIQ()
	require.Equal(t, xml.ResultType, result.Type())
	require.Equal(t, id, result.ID())
}

func TestIQJID(t *testing.T) {
	from, _ := jid.New("ortuman", "test.org", "balcony", false)
	to, _ := jid.New("ortuman", "example.org", "garden", false)
	iq := xml.NewIQType(uuid.New(), xml.GetType)
	iq.SetFromJID(from)
	require.Equal(t, iq.FromJID().String(), iq.From())
	iq.SetToJID(to)
	require.Equal(t, iq.ToJID().String(), iq.To())
}
