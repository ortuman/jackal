/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import "testing"

func TestElementNameAndNamespace(t *testing.T) {
	e := NewElementNamespace("iq", "jabber:client")
	if e.Name() != "iq" {
		t.Errorf("name %s. expected %s", e.Name(), "iq")
	}
	if e.Namespace() != "jabber:client" {
		t.Errorf("namespace %s. expected %s", e.Namespace(), "jabber:client")
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
		t.Error("e1.p == e2.p after setting name")
	}
	e3.SetText("another text")
	if e1.shared() == e3.shared() {
		t.Error("e1.p == e3.p after setting text")
	}
	e4.SetAttribute("id", "abcde")
	if e1.shared() == e4.shared() {
		t.Error("e1.p == e4.p after setting attribute")
	}
	e5.AppendElement(NewElementName("item"))
	if e1.shared() == e5.shared() {
		t.Error("e1.p == e5.p after appending element")
	}
}
