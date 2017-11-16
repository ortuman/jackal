/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import "testing"

func TestElementNameAndNamespace(t *testing.T) {
	e := NewElementNamespace("iq", "jabber:client")
	if e.Name() != "iq" {
		t.Fatalf("name %s. expected %s", e.Name(), "iq")
	}
	if e.Namespace() != "jabber:client" {
		t.Fatalf("namespace %s. expected %s", e.Namespace(), "jabber:client")
	}
}

func TestAttribute(t *testing.T) {
}

func TestAppendElement(t *testing.T) {
	e1 := NewElementNamespace("iq", "jabber:client")
	q := NewElementNamespace("query", "im.jackal")
	e1.AppendElement(q)

	q1 := e1.Element("query")
	t.Log(q1)
	if q1 == nil {
		t.Fatal("q1 not found")
	}
	q2 := e1.ElementNamespace("query", "im.jackal")
	if q2 == nil || q2.shared() != q1.shared() {
		t.Fatal("q2 not found")
	}
}

func TestShadowCopy(t *testing.T) {
	e1 := NewElementNamespace("iq", "jabber:client")
	e1.AppendElement(NewElementNamespace("query", "im.jackal"))
	e1.SetText("a text")

	e2 := Element{e1.p, 0}
	e3 := Element{e1.p, 0}
	e4 := Element{e1.p, 0}
	e5 := Element{e1.p, 0}

	e2.SetName("message")
	if e1.shared() == e2.shared() {
		t.Fatal("e1.p == e2.p after setting name")
	}
	e3.SetText("another text")
	if e1.shared() == e3.shared() {
		t.Fatal("e1.p == e3.p after setting text")
	}
	e4.SetAttribute("id", "abcde")
	if e1.shared() == e4.shared() {
		t.Fatal("e1.p == e4.p after setting attribute")
	}
	e5.AppendElement(NewElementName("item"))
	if e1.shared() == e5.shared() {
		t.Fatal("e1.p == e5.p after appending element")
	}
}
