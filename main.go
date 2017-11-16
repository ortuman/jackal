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
	println(e.XML(true))
}
