/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

// MutableElement represents a mutable XML node element.
type MutableElement struct {
	xElement
}

// NewElementName creates a mutable XML Element instance with a given name.
func NewElementName(name string) *MutableElement {
	m := &MutableElement{}
	m.name = name
	return m
}

// NewElementNamespace creates a mutable XML Element instance with a given name and namespace.
func NewElementNamespace(name, namespace string) *MutableElement {
	m := &MutableElement{}
	m.name = name
	m.attrs = AttributeSet{attrs: []Attribute{{"xmlns", namespace}}}
	return m
}

// NewElementFromElement creates a mutable XML Element by copying an element.
func NewElementFromElement(elem Element) *MutableElement {
	m := &MutableElement{}
	m.copyFrom(elem)
	return m
}

// SetNamespace sets 'xmlns' node attribute.
func (m *MutableElement) SetNamespace(namespace string) {
	m.SetAttribute("xmlns", namespace)
}

// SetID sets 'id' node attribute.
func (m *MutableElement) SetID(identifier string) {
	m.SetAttribute("id", identifier)
}

// SetType sets 'type' node attribute.
func (m *MutableElement) SetType(tp string) {
	m.SetAttribute("type", tp)
}

// SetLanguage sets 'xml:lang' node attribute.
func (m *MutableElement) SetLanguage(language string) {
	m.SetAttribute("xml:lang", language)
}

// SetVersion sets 'version' node attribute.
func (m *MutableElement) SetVersion(version string) {
	m.SetAttribute("version", version)
}

// SetFrom sets 'from' node attribute.
func (m *MutableElement) SetFrom(from string) {
	m.SetAttribute("from", from)
}

// SetTo sets 'to' node attribute.
func (m *MutableElement) SetTo(to string) {
	m.SetAttribute("to", to)
}

// SetName sets XML node name.
func (m *MutableElement) SetName(name string) {
	m.name = name
}

// SetText sets XML node text value.
func (m *MutableElement) SetText(text string) {
	m.text = text
}

// SetAttribute sets an XML node attribute (label=value)
func (m *MutableElement) SetAttribute(label, value string) {
	m.attrs.setAttribute(label, value)
}

// RemoveAttribute removes an XML node attribute.
func (m *MutableElement) RemoveAttribute(label string) {
	m.attrs.removeAttribute(label)
}

// AppendElement appends a new sub element.
func (m *MutableElement) AppendElement(element Element) {
	m.appendElement(element)
}

// AppendElements appends an array of sub elements.
func (m *MutableElement) AppendElements(elements []Element) {
	m.appendElements(elements)
}

// RemoveElements removes all elements with a given name.
func (m *MutableElement) RemoveElements(name string) {
	m.removeElements(name)
}

// RemoveElementsNamespace removes all elements with a given name and namespace.
func (m *MutableElement) RemoveElementsNamespace(name, namespace string) {
	m.removeElementsNamespace(name, namespace)
}

// ClearElements removes all elements.
func (m *MutableElement) ClearElements() {
	m.elements = nil
}
