/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

// MutableElement represents a mutable XML node element.
// This type adds mutable operations to the basic behavior inherited from Element.
type MutableElement struct {
	Element
}

// NewMutableElement creates a MutableElement instance from an immutable element.
func NewMutableElement(e *Element) *MutableElement {
	m := MutableElement{}
	m.name = e.name
	m.text = e.text
	m.attrs = make([]Attribute, len(e.attrs), cap(e.attrs))
	m.elements = make([]*Element, len(e.elements), cap(e.elements))
	copy(m.attrs, e.attrs)
	copy(m.elements, e.elements)
	return &m
}

// NewMutableElementName creates MutableElement instance with a given name.
func NewMutableElementName(name string) *MutableElement {
	m := &MutableElement{}
	m.name = name
	m.attrs = []Attribute{}
	m.elements = []*Element{}
	return m
}

// NewMutableElementNamespace creates MutableElement instance with a given name and namespace.
func NewMutableElementNamespace(name, namespace string) *MutableElement {
	m := &MutableElement{}
	m.name = name
	m.attrs = []Attribute{{"xmlns", namespace}}
	m.elements = []*Element{}
	return m
}

// SetAttribute sets an XML node attribute (label=value)
func (m *MutableElement) SetAttribute(label, value string) {
	for i := 0; i < len(m.attrs); i++ {
		if m.attrs[i].label == label {
			m.attrs[i].value = value
			return
		}
	}
	m.attrs = append(m.attrs, Attribute{label, value})
}

// RemoveAttribute removes an XML node attribute.
func (m *MutableElement) RemoveAttribute(label string) {
	for i := 0; i < len(m.attrs); i++ {
		if m.attrs[i].label == label {
			m.attrs = append(m.attrs[:i], m.attrs[i+1:]...)
			return
		}
	}
}

// AppendElement appends a new subelement.
func (m *MutableElement) AppendElement(element *Element) {
	m.elements = append(m.elements, element)
}

// AppendElements appends an array of elements.
func (m *MutableElement) AppendElements(elements []*Element) {
	m.elements = append(m.elements, elements...)
}

// RemoveElements removes all elements with a given name.
func (m *MutableElement) RemoveElements(name string) {
	filtered := m.elements[:0]
	for _, e := range m.elements {
		if e.name != name {
			filtered = append(filtered, e)
		}
	}
	m.elements = filtered
}

// RemoveElementsNS removes all elements with a given name and namespace.
func (m *MutableElement) RemoveElementsNS(name, namespace string) {
	filtered := m.elements[:0]
	for _, e := range m.elements {
		if e.name != name || e.Namespace() != namespace {
			filtered = append(filtered, e)
		}
	}
	m.elements = filtered
}

// ClearElements removes all elements.
func (m *MutableElement) ClearElements() {
	m.elements = []*Element{}
}

// MutableCopy returns a new instance that’s an mutable copy of the receiver.
func (m *MutableElement) MutableCopy() *MutableElement {
	cp := &MutableElement{}
	cp.name = m.name
	cp.text = m.text
	cp.attrs = make([]Attribute, len(m.attrs), cap(m.attrs))
	cp.elements = make([]*Element, len(m.elements), cap(m.elements))
	copy(cp.attrs, m.attrs)
	copy(cp.elements, m.elements)
	return cp
}
