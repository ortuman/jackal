/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

// SetID sets 'id' node attribute.
func (e *Element) SetID(identifier string) {
	e.SetAttribute("id", identifier)
}

// ID returns 'id' node attribute.
func (e *Element) ID() string {
	return e.Attribute("id")
}

// SetNamespace sets 'xmlns' node attribute.
func (e *Element) SetNamespace(namespace string) {
	e.SetAttribute("xmlns", namespace)
}

// Namespace returns 'xmlns' node attribute.
func (e *Element) Namespace() string {
	return e.Attribute("xmlns")
}

// SetLanguage sets 'xml:lang' node attribute.
func (e *Element) SetLanguage(language string) {
	e.SetAttribute("xml:lang", language)
}

// Language returns 'xml:lang' node attribute.
func (e *Element) Language() string {
	return e.Attribute("xml:lang")
}

// SetVersion sets 'version' node attribute.
func (e *Element) SetVersion(version string) {
	e.SetAttribute("version", version)
}

// Version returns 'version' node attribute.
func (e *Element) Version() string {
	return e.Attribute("version")
}
