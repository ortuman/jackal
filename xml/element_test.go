/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml_test

import (
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/assert"
)

func TestElementNameAndNamespace(t *testing.T) {
	e := xml.NewElementNamespace("n", "ns")
	assert.Equal(t, e.Name(), "n")
	assert.Equal(t, e.Namespace(), "ns")
}

func TestAttribute(t *testing.T) {
	e := xml.NewElementName("n")
	e.SetID("123")
	e.SetLanguage("en")
	e.SetVersion("1.0")

	assert.Equal(t, e.ID(), "123")
	assert.Equal(t, e.Language(), "en")
	assert.Equal(t, e.Version(), "1.0")
}

func TestElement(t *testing.T) {
	e := xml.NewElementName("n")
	e.AppendElement(xml.NewElementName("a"))
	e.AppendElement(xml.NewElementName("b"))
	e.AppendElement(xml.NewElementNamespace("c", "ns1"))
	e.AppendElement(xml.NewElementNamespace("c", "ns2"))
	e.AppendElement(xml.NewElementNamespace("c", "ns3"))
	e.AppendElement(xml.NewElementName("d"))
	a := e.FindElement("a")
	assert.NotNil(t, a)

	c := e.FindElements("c")
	assert.Equal(t, len(c), 3)

	c1 := e.FindElementsNamespace("c", "ns1")
	assert.Equal(t, len(c1), 1)
	assert.Equal(t, e.ElementsCount(), 6)
}
