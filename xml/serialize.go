/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

// XML converts an Element entity to its raw XML representation.
// If includeClosing is true a closing tag will be attached.
func (e *Element) XML(includeClosing bool) string {
	ret := "<" + e.name

	// serialize attributes
	for i := 0; i < len(e.attrs); i++ {
		if len(e.attrs[i].value) == 0 {
			continue
		}
		ret += " " + e.attrs[i].label + "=\"" + e.attrs[i].value + "\""
	}
	if len(e.childs) > 0 || len(e.text) > 0 {
		ret += ">"

		// serialize text
		if len(e.text) > 0 {
			ret += e.text
		}
		// serialize child elements
		for j := 0; j < len(e.childs); j++ {
			ret += e.childs[j].XML(true)
		}
		if includeClosing {
			ret += "</" + e.name + ">"
		}
	} else {
		if includeClosing {
			ret += "/>"
		} else {
			ret += ">"
		}
	}
	return ret
}
