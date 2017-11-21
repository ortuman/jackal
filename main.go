/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package main

import (
	"fmt"
	"strings"

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

	/*
		docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<a xmlns="im.jackal.a">` +
			"       So they say\n     LA\nLOLA     \t" +
			`</a>\n`
	*/
	docSrc := `<?xml version="1.0"?>` +
		`<stream:stream xmlns:stream="http://etherx.jabber.org/streams" version="1.0" xmlns="jabber:client" to="localhost" xml:lang="en" xmlns:xml="http://www.w3.org/XML/1998/namespace">`

	p := xml.NewParser()
	err := p.ParseElements(strings.NewReader(docSrc))
	if err != nil {
		fmt.Printf("%v", err)
	} else {
		e := p.PopElement()
		if e != nil {
			fmt.Printf("%s", e.XML(true))
		}
	}
	// println(e.Delayed("im.jackal", "Offline storage").XML(true))
}
