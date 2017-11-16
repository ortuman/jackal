/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import "testing"

func TestElementNameAndNamespace(t *testing.T) {
	e := NewElementNamespace("n", "ns")
	if e.Name() != "n" {
		t.Fatalf("wrong name: '%s'. expected 'n'", e.Name())
	}
	if e.Namespace() != "ns" {
		t.Fatalf("wrong namespace: '%s'. expected 'ns'", e.Namespace())
	}
}

func TestAttribute(t *testing.T) {
	e := NewMutableElementName("n")
	e.SetID("123")
	e.SetLanguage("en")
	e.SetVersion("1.0")

	if e.ID() != "123" {
		t.Fatalf("id == %s. expected 123", e.ID())
	}
	if e.AttributesCount() != 3 {
		t.Fatalf("attributes count == %d. expected 3.", e.AttributesCount())
	}
	e.RemoveAttribute("xml:lang")
	if e.AttributesCount() != 2 {
		t.Fatalf("attributes count == %d. expected 2.", e.AttributesCount())
	}
}

func TestElement(t *testing.T) {
	e := NewMutableElementName("n")
	e.AppendElement(NewElementName("a"))
	e.AppendElement(NewElementName("b"))
	e.AppendElement(NewElementNamespace("c", "ns1"))
	e.AppendElement(NewElementNamespace("c", "ns2"))
	e.AppendElement(NewElementNamespace("c", "ns3"))
	e.AppendElement(NewElementName("d"))
	a := e.FindElement("a")
	if a == nil {
		t.Fatal("a == nil")
	}
	c := e.FindElements("c")
	if len(c) != 3 {
		t.Fatalf("len(c) != %d. expected 3", len(c))
	}
	c1 := e.FindElementsNamespace("c", "ns1")
	if len(c1) != 1 {
		t.Fatalf("len(c1) == %d. expected 1", len(c1))
	}
}
