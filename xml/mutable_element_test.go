/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ortuman/jackal/xml"
)

func TestSetName(t *testing.T) {
	m := xml.NewMutableElementName("n")
	m.SetName("n2")
	assert.Equal(t, m.Name(), "n2")
}

func TestSetText(t *testing.T) {
	m := xml.NewMutableElementName("n")
	m.SetText("example text")
	assert.Equal(t, m.Text(), "example text")
}

func TestSetAttribute(t *testing.T) {
	m := xml.NewMutableElementName("n")
	m.SetAttribute("att", "val")
	assert.Equal(t, m.Attribute("att"), "val")

	// replace attribute
	m.SetAttribute("att", "val2")
	assert.Equal(t, m.Attribute("att"), "val2")
}

func TestRemoveAttribute(t *testing.T) {
	m := xml.NewMutableElementName("n")
	m.SetAttribute("att", "val")
	m.SetAttribute("att2", "val2")
	assert.Equal(t, m.AttributesCount(), 2)

	m.RemoveAttribute("att")
	assert.Equal(t, m.AttributesCount(), 1)
	assert.Empty(t, m.Attribute("att"))
	assert.Equal(t, m.Attribute("att2"), "val2")

	m.RemoveAttribute("att2")
	assert.Equal(t, m.AttributesCount(), 0)
}

func TestRemoveElements(t *testing.T) {
	m := xml.NewMutableElementName("n")
	m.AppendElement(xml.NewElementName("a"))
	m.AppendElement(xml.NewElementNamespace("b", "ns1"))
	m.AppendElement(xml.NewElementNamespace("b", "ns2"))
	m.AppendElement(xml.NewElementNamespace("b", "ns3"))
	m.AppendElement(xml.NewElementName("c"))
	assert.Equal(t, m.ElementsCount(), 5)

	m.RemoveElementsNamespace("b", "ns1")
	assert.Equal(t, m.ElementsCount(), 4)

	b2 := m.FindElementNamespace("b", "ns2")
	assert.NotNil(t, b2)

	m.RemoveElements("b")
	assert.Equal(t, m.ElementsCount(), 2)

	m.ClearElements()
	assert.Equal(t, m.ElementsCount(), 0)
}

func TestMutableCopy(t *testing.T) {
	m := xml.NewMutableElementName("n")
	m.SetAttribute("att", "val")
	m.SetText("a text")
	m.AppendElement(xml.NewElementName("a"))
	cp := m.MutableCopy()

	assert.NotEqual(t, cp.ElementsCount(), 0)
	assert.NotEqual(t, m.ElementsCount(), 0)
	assert.Equal(t, cp.Name(), m.Name())
	assert.Equal(t, cp.Text(), m.Text())
	assert.Equal(t, cp.AttributesCount(), m.AttributesCount())
	assert.Equal(t, cp.Attribute("att"), m.Attribute("att"))
	assert.Equal(t, cp.ElementsCount(), m.ElementsCount())
	assert.Equal(t, cp.Elements()[0], m.Elements()[0])
}
