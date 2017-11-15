/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

// XML converts an Element entity to its raw XML representation.
// If includeClosing is true a closing tag will be attached.
func (e *Element) XML(includeClosing bool) string {
	s := e.shared()
	ret := "<" + s.name

	// serialize attributes
	for i := 0; i < len(s.attributes); i++ {
		if len(s.attributes[i].value) == 0 {
			continue
		}
		ret += " " + s.attributes[i].label + "=\"" + s.attributes[i].value + "\""
	}
	if len(s.childElements) > 0 || len(s.text) > 0 {
		ret += ">"

		// serialize text
		if len(s.text) > 0 {
			ret += s.text
		}
		// serialize child elements
		for j := 0; j < len(s.childElements); j++ {
			ret += s.childElements[j].XML(true)
		}
		if includeClosing {
			ret += "</" + s.name + ">"
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
