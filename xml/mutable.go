/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

// SetName sets XML node name.
func (e *XElement) SetName(name string) {
	e.name = name
}

// SetAttribute sets an XML node attribute (label=value)
func (e *XElement) SetAttribute(label, value string) {
	e.attrs.setAttribute(label, value)
}

// RemoveAttribute removes an XML node attribute.
func (e *XElement) RemoveAttribute(label string) {
	e.attrs.removeAttribute(label)
}

// AppendElement appends a new sub element.
func (e *XElement) AppendElement(element Element) {
	e.elements.append(element)
}

// AppendElements appends an array of sub elements.
func (e *XElement) AppendElements(elements []Element) {
	e.elements.append(elements...)
}

// RemoveElements removes all elements with a given name.
func (e *XElement) RemoveElements(name string) {
	e.elements.remove(name)
}

// RemoveElementsNamespace removes all elements with a given name and namespace.
func (e *XElement) RemoveElementsNamespace(name, namespace string) {
	e.elements.removeNamespace(name, namespace)
}

// SetNamespace sets 'xmlns' node attribute.
func (e *XElement) SetNamespace(namespace string) {
	e.attrs.setAttribute("xmlns", namespace)
}

// ClearElements removes all elements.
func (e *XElement) ClearElements() {
	e.elements.clear()
}

// SetText sets XML node text value.
func (e *XElement) SetText(text string) {
	e.text = text
}

// SetID sets 'id' node attribute.
func (e *XElement) SetID(identifier string) {
	e.attrs.setAttribute("id", identifier)
}

// SetLanguage sets 'xml:lang' node attribute.
func (e *XElement) SetLanguage(language string) {
	e.attrs.setAttribute("xml:lang", language)
}

// SetFrom sets 'from' node attribute.
func (e *XElement) SetFrom(from string) {
	e.attrs.setAttribute("from", from)
}

// SetTo sets 'to' node attribute.
func (e *XElement) SetTo(to string) {
	e.attrs.setAttribute("to", to)
}

// SetType sets 'type' node attribute.
func (m *XElement) SetType(tp string) {
	m.attrs.setAttribute("type", tp)
}
