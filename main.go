/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package main

import (
	"github.com/ortuman/jackal/xml"
)

func main() {
	e := xml.NewElementNS("iq", "jabber:client")
	println(e.XML(true))
}
