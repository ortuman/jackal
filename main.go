/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package main

import (
	"github.com/ortuman/jackal/xml"
)

func main() {
	e := xml.NewMutableElementNamespace("iq", "jabber:client")
	e.SetID("123")
	e.SetLanguage("en")
	e.SetVersion("1.0")

	e.AppendElement(xml.NewElementName("a"))
	e.AppendElement(xml.NewElementName("b"))
	e.AppendElement(xml.NewElementName("c"))
	e.AppendElement(xml.NewElementName("c"))
	e.AppendElement(xml.NewElementName("c"))
	e.AppendElement(xml.NewElementName("d"))
	e.AppendElement(xml.NewElementName("e"))

	e.RemoveElements("c")
	e.RemoveElements("e")
	e.RemoveElements("a")

	println(e.Delayed("im.jackal", "Offline storage").XML(true))
}
