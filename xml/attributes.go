/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

// SetID sets 'id' node attribute.
func (e *XElement) SetID(identifier string) {
	e.SetAttribute("id", identifier)
}

// ID returns 'id' node attribute.
func (e *XElement) ID() string {
	return e.Attribute("id")
}

// SetNamespace sets 'xmlns' node attribute.
func (e *XElement) SetNamespace(namespace string) {
	e.SetAttribute("xmlns", namespace)
}

// Namespace returns 'xmlns' node attribute.
func (e *XElement) Namespace() string {
	return e.Attribute("xmlns")
}

// SetLanguage sets 'xml:lang' node attribute.
func (e *XElement) SetLanguage(language string) {
	e.SetAttribute("xml:lang", language)
}

// Language returns 'xml:lang' node attribute.
func (e *XElement) Language() string {
	return e.Attribute("xml:lang")
}

// SetVersion sets 'version' node attribute.
func (e *XElement) SetVersion(version string) {
	e.SetAttribute("version", version)
}

// Version returns 'version' node attribute.
func (e *XElement) Version() string {
	return e.Attribute("version")
}

// SetFrom sets 'from' node attribute.
func (e *XElement) SetFrom(from string) {
	e.SetAttribute("from", from)
}

// From returns 'from' node attribute.
func (e *XElement) From() string {
	return e.Attribute("from")
}

// SetTo sets 'to' node attribute.
func (e *XElement) SetTo(to string) {
	e.SetAttribute("to", to)
}

// To returns 'to' node attribute.
func (e *XElement) To() string {
	return e.Attribute("to")
}

// Type returns 'type' node attribute.
func (e *XElement) Type() string {
	return e.Attribute("type")
}

// SetType sets 'type' node attribute.
func (e *XElement) SetType(tp string) {
	e.SetAttribute("type", tp)
}
