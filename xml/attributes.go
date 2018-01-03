/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

// SetID sets 'id' node attribute.
func (e *MutableElement) SetID(identifier string) {
	e.SetAttribute("id", identifier)
}

// ID returns 'id' node attribute.
func (e *Element) ID() string {
	return e.Attribute("id")
}

// SetNamespace sets 'xmlns' node attribute.
func (e *MutableElement) SetNamespace(namespace string) {
	e.SetAttribute("xmlns", namespace)
}

// Namespace returns 'xmlns' node attribute.
func (e *Element) Namespace() string {
	return e.Attribute("xmlns")
}

// SetLanguage sets 'xml:lang' node attribute.
func (e *MutableElement) SetLanguage(language string) {
	e.SetAttribute("xml:lang", language)
}

// Language returns 'xml:lang' node attribute.
func (e *Element) Language() string {
	return e.Attribute("xml:lang")
}

// SetVersion sets 'version' node attribute.
func (e *MutableElement) SetVersion(version string) {
	e.SetAttribute("version", version)
}

// Version returns 'version' node attribute.
func (e *Element) Version() string {
	return e.Attribute("version")
}

// SetFrom sets 'from' node attribute.
func (e *MutableElement) SetFrom(from string) {
	e.SetAttribute("from", from)
}

// From returns 'from' node attribute.
func (e *Element) From() string {
	return e.Attribute("from")
}

// SetTo sets 'to' node attribute.
func (e *MutableElement) SetTo(to string) {
	e.SetAttribute("to", to)
}

// To returns 'to' node attribute.
func (e *Element) To() string {
	return e.Attribute("to")
}

// Type returns 'type' node attribute.
func (e *Element) Type() string {
	return e.Attribute("type")
}

// SetType sets 'type' node attribute.
func (e *MutableElement) SetType(tp string) {
	e.SetAttribute("type", tp)
}
