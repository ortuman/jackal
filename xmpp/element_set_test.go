/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestElementSet_Children(t *testing.T) {
	var es elementSet
	a1 := NewElementName("a")
	a2 := NewElementName("a")
	els := []XElement{a1, a2}
	es.append(els...)
	require.Equal(t, els, es.Children("a"))
	require.Equal(t, a1, es.Child("a"))
	require.Nil(t, es.Child("d"))
}

func TestElementSet_ChildrenNamespace(t *testing.T) {
	var es elementSet
	a1 := NewElementNamespace("a", "ns1")
	a2 := NewElementNamespace("a", "ns1")
	a3 := NewElementNamespace("a", "ns2")
	els0 := []XElement{a1, a2}
	els1 := []XElement{a1, a2, a3}
	es.append(els1...)
	require.Equal(t, els0, es.ChildrenNamespace("a", "ns1"))
	require.Equal(t, a1, es.ChildNamespace("a", "ns1"))
	require.Nil(t, es.ChildNamespace("d", "ns1"))
}

func TestElementSet_All(t *testing.T) {
	var es elementSet
	a1 := NewElementNamespace("a", "ns1")
	a2 := NewElementNamespace("a", "ns1")
	a3 := NewElementNamespace("a", "ns2")
	els := []XElement{a1, a2, a3}
	es.append(els...)
	require.Equal(t, els, es.All())
	require.Equal(t, 3, es.Count())
}

func TestElementSet_Remove(t *testing.T) {
	var es elementSet
	a1 := NewElementNamespace("a", "ns1")
	a2 := NewElementNamespace("b", "ns1")
	a3 := NewElementNamespace("a", "ns2")
	els0 := []XElement{a1}
	els1 := []XElement{a1, a2}
	els2 := []XElement{a1, a2, a3}
	es.append(els2...)
	es.removeNamespace("a", "ns2")
	require.Equal(t, els1, es.All())
	es.remove("b")
	require.Equal(t, els0, es.All())
	es.clear()
	require.Equal(t, 0, len(es.All()))
}

func TestElementSet_Copy(t *testing.T) {
	var es0, es1 elementSet
	a1 := NewElementNamespace("a", "ns1")
	a2 := NewElementNamespace("b", "ns1")
	a3 := NewElementNamespace("a", "ns2")
	es0.append(a1, a2, a3)
	es1.copyFrom(es0)
	require.Equal(t, es0.Count(), es1.Count())
	for i, el := range es0 {
		require.Equal(t, el.String(), es1[i].String())
	}
}

func TestElementSet_Gob(t *testing.T) {
	var es0, es1 elementSet
	a1 := NewElementNamespace("a", "ns1")
	a2 := NewElementNamespace("b", "ns1")
	a3 := NewElementNamespace("c", "ns2")
	es0.append(a1, a2, a3)

	buf := new(bytes.Buffer)
	require.Nil(t, es0.ToBytes(buf))
	require.Nil(t, es1.FromBytes(buf))
	require.Equal(t, es0.Count(), es1.Count())
	for i, el := range es0 {
		require.Equal(t, el.String(), es1[i].String())
	}
}
