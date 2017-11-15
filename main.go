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
	println(e.Attribute("id"))
}
