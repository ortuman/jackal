/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml_test

import (
	"bytes"
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestElementNameAndNamespace(t *testing.T) {
	e := xml.NewElementNamespace("n", "ns")
	require.Equal(t, "n", e.Name())
	require.Equal(t, "ns", e.Namespace())
	e.SetName("e2")
	require.Equal(t, "e2", e.Name())
}

func TestAttributes(t *testing.T) {
	e := xml.NewElementName("elem")

	// test attribute setters/getters
	e.SetNamespace("ns")
	require.Equal(t, "ns", e.Namespace())

	e.SetID("123")
	require.Equal(t, "123", e.ID())

	e.SetLanguage("en")
	require.Equal(t, "en", e.Language())

	e.SetVersion("1.0")
	require.Equal(t, "1.0", e.Version())

	e.SetFrom("ortuman@example.org")
	require.Equal(t, "ortuman@example.org", e.From())

	e.SetTo("ortuman@example.org")
	require.Equal(t, "ortuman@example.org", e.To())

	e.SetType("chat")
	require.Equal(t, "chat", e.Type())

	require.Equal(t, "", e.Attributes().Get("not_existing"))

	require.Equal(t, 7, e.Attributes().Len())
	require.Equal(t, 7, e.Attributes().Len())

	// replace attribute
	e.SetType("error")
	require.True(t, e.IsError())

	// remove attribute
	e.RemoveAttribute("type")
	require.Equal(t, "", e.Type())
}

func TestText(t *testing.T) {
	e := xml.NewElementName("elem")
	e.SetText("This is a sample text Ñ")
	require.Equal(t, "This is a sample text Ñ", e.Text())
	require.Equal(t, 24, len(e.Text()))
}

func TestChildElement(t *testing.T) {
	e := xml.NewElementName("n")
	e.AppendElement(xml.NewElementName("a"))
	e.AppendElement(xml.NewElementName("b"))
	e.AppendElement(xml.NewElementNamespace("c", "ns1"))
	e.AppendElement(xml.NewElementNamespace("c", "ns2"))
	e.AppendElement(xml.NewElementNamespace("c", "ns3"))
	e.AppendElement(xml.NewElementName("d"))

	a := e.FindElement("a")
	require.NotNil(t, a)

	c0 := e.FindElementNamespace("c", "ns3")
	require.NotNil(t, c0)

	c1 := e.FindElements("c")
	require.Equal(t, 3, len(c1))

	c2 := e.FindElementsNamespace("c", "ns1")
	require.Equal(t, 1, len(c2))
	require.Equal(t, 6, e.ElementsCount())

	e.RemoveElementsNamespace("c", "ns1")
	c3 := e.FindElementNamespace("c", "ns1")
	require.Nil(t, c3)

	e.RemoveElements("c")
	c1 = e.FindElements("c")
	require.Nil(t, c1)

	c4 := e.FindElementNamespace("c", "ns5")
	require.Nil(t, c4)

	z := e.FindElement("z")
	require.Nil(t, z)

	cs := e.Elements()
	require.Equal(t, 3, len(cs))
	e.ClearElements()
	require.Nil(t, e.Elements())
	e.AppendElements(cs)
	require.Equal(t, 3, e.ElementsCount())
}

func TestCopy(t *testing.T) {
	e := xml.NewElementName("elem")
	e.SetNamespace("ns1")
	e.SetID(uuid.New())
	e.SetText("A simple text")
	e.AppendElement(xml.NewElementName("child1"))
	e.AppendElement(xml.NewElementName("child2"))

	cp := e.Copy()
	require.Equal(t, e.String(), cp.String())
}

func TestString(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	e := xml.NewElementName("elem")

	e.ToXML(buf, false)
	require.Equal(t, "<elem>", buf.String())

	buf.Reset()
	e.ToXML(buf, true)
	require.Equal(t, "<elem/>", buf.String())

	e.AppendElements([]xml.Element{xml.NewElementName("child1"), xml.NewElementName("child2")})
	buf.Reset()
	e.ToXML(buf, true)
	require.Equal(t, "<elem><child1/><child2/></elem>", buf.String())

	e.SetType("normal")
	e.SetID("")
	buf.Reset()
	e.ToXML(buf, true)
	require.Equal(t, `<elem type="normal"><child1/><child2/></elem>`, buf.String())
}
