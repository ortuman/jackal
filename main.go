/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package main

import (
	"github.com/ortuman/jackal/xml"
)

func main() {
	e := xml.NewElementNamespace("iq", "github.com")
	e.SetAttribute("id", "123")
	e.SetAttribute("id", "456")
	e.SetNamespace("jabber:client")

	e1 := xml.NewElementName("success")
	e1.SetText("a sucessful text")
	e.AppendElement(e1)

	s := e.ElementNamespace("success", "kk")
	println(s)

	println(e.XML(true))
}
