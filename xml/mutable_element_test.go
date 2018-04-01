/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestElement_RemoveAttribute(t *testing.T) {
	e := NewElementNamespace("n", "ns")
	require.Equal(t, "ns", e.Attributes().Get("xmlns"))
	e.RemoveAttribute("xmlns")
	require.Equal(t, "", e.Attributes().Get("xmlns"))
}

func TestElement_RemoveElements(t *testing.T) {
	e := NewElementName("n")
	e.AppendElement(NewElementNamespace("a", "ns1"))
	e.AppendElement(NewElementNamespace("a", "ns2"))
	e.AppendElement(NewElementNamespace("a", "ns3"))
	e.AppendElement(NewElementName("b"))
	e.AppendElement(NewElementName("c"))
	require.Equal(t, 5, e.Elements().Count())
	e.RemoveElementsNamespace("a", "ns3")
	require.Equal(t, 4, e.Elements().Count())
	e.RemoveElements("a")
	require.Equal(t, 2, e.Elements().Count())
	e.ClearElements()
	require.Equal(t, 0, e.Elements().Count())
}
