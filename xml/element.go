/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"bytes"
	"sync"
	"unicode/utf8"
)

// Attribute represents an XML node attribute (label=value).
type Attribute struct {
	Label string
	Value string
}

// Serializable is an interface type. A Serializable entity describes a value
// that could be serialized to a raw XML representation.
// includeClosing determines if closing tag should be attached.
type Serializable interface {
	XML(includeClosing bool) string
}

// Element represents an XML node element.
type Element interface {
	Serializable

	// Name returns XML node name.
	Name() string

	// Text returns XML node text value.
	// Returns an empty string if not set.
	Text() string

	// TextLen returns XML node text value length.
	TextLen() int

	// Attribute returns XML node attribute value.
	Attribute(label string) string

	// Attributes returns all XML node attributes.
	Attributes() []Attribute

	// AttributesCount XML attributes count.
	AttributesCount() int

	// FindElement returns first element identified by name.
	// Returns nil if no element is found.
	FindElement(name string) Element

	// FindElements returns all elements identified by name.
	// Returns an empty array if no elements are found.
	FindElements(name string) []Element

	// FindElementNamespace returns first element identified by name and namespace.
	// Returns nil if no element is found.
	FindElementNamespace(name, namespace string) Element

	// FindElementsNamespace returns all elements identified by name and namespace.
	// Returns an empty array if no elements are found.
	FindElementsNamespace(name, namespace string) []Element

	// Elements returns all instance's child elements.
	Elements() []Element

	// ElementsCount returns child elements count.
	ElementsCount() int
}

var serializeBufs = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

type XElement struct {
	name     string
	text     string
	attrs    []Attribute
	elements []Element
}

// NewElementName creates an XML Element instance with a given name.
func NewElementName(name string) *XElement {
	return &XElement{
		name: name,
	}
}

// NewElementAttributes creates an XML Element instance with a given name and attributes.
func NewElementAttributes(name string, attributes []Attribute) *XElement {
	return &XElement{
		name:  name,
		attrs: attributes,
	}
}

// NewElementNamespace creates an XML Element instance with a given name and namespace.
func NewElementNamespace(name, namespace string) *XElement {
	return NewElementAttributes(name, []Attribute{{"xmlns", namespace}})
}

func (e *XElement) Name() string {
	return e.name
}

func (e *XElement) Text() string {
	return e.text
}

func (e *XElement) TextLen() int {
	return utf8.RuneCountInString(e.text)
}

func (e *XElement) Attribute(label string) string {
	for i := 0; i < len(e.attrs); i++ {
		if e.attrs[i].Label == label {
			return e.attrs[i].Value
		}
	}
	return ""
}

func (e *XElement) Attributes() []Attribute {
	return e.attrs
}

func (e *XElement) AttributesCount() int {
	return len(e.attrs)
}

func (e *XElement) FindElement(name string) Element {
	for i := 0; i < len(e.elements); i++ {
		if e.elements[i].Name() == name {
			return e.elements[i]
		}
	}
	return nil
}

func (e *XElement) FindElements(name string) []Element {
	ret := e.elements[:0]
	for i := 0; i < len(e.elements); i++ {
		if e.elements[i].Name() == name {
			ret = append(ret, e.elements[i])
		}
	}
	return ret
}

func (e *XElement) FindElementNamespace(name, namespace string) Element {
	for i := 0; i < len(e.elements); i++ {
		if e.elements[i].Name() == name && e.elements[i].Attribute("xmlns") == namespace {
			return e.elements[i]
		}
	}
	return nil
}

func (e *XElement) FindElementsNamespace(name, namespace string) []Element {
	ret := e.elements[:0]
	for i := 0; i < len(e.elements); i++ {
		if e.elements[i].Name() == name && e.elements[i].Attribute("xmlns") == namespace {
			ret = append(ret, e.elements[i])
		}
	}
	return ret
}

func (e *XElement) Elements() []Element {
	return e.elements
}

func (e *XElement) ElementsCount() int {
	return len(e.elements)
}

// SetName sets XML node name.
func (e *XElement) SetName(name string) {
	e.name = name
}

// SetText sets XML node text value.
func (e *XElement) SetText(text string) {
	e.text = text
}

// SetAttribute sets an XML node attribute (label=value)
func (e *XElement) SetAttribute(label, value string) {
	for i := 0; i < len(e.attrs); i++ {
		if e.attrs[i].Label == label {
			e.attrs[i].Value = value
			return
		}
	}
	e.attrs = append(e.attrs, Attribute{label, value})
}

// RemoveAttribute removes an XML node attribute.
func (e *XElement) RemoveAttribute(label string) {
	for i := 0; i < len(e.attrs); i++ {
		if e.attrs[i].Label == label {
			e.attrs = append(e.attrs[:i], e.attrs[i+1:]...)
			return
		}
	}
}

// AppendElement appends a new sub element.
func (e *XElement) AppendElement(element Element) {
	e.elements = append(e.elements, element)
}

// AppendElements appends an array of sub elements.
func (e *XElement) AppendElements(elements ...Element) {
	e.elements = append(e.elements, elements...)
}

// RemoveElements removes all elements with a given name.
func (e *XElement) RemoveElements(name string) {
	filtered := e.elements[:0]
	for _, elem := range e.elements {
		if elem.Name() != name {
			filtered = append(filtered, elem)
		}
	}
	e.elements = filtered
}

// RemoveElementsNamespace removes all elements with a given name and namespace.
func (e *XElement) RemoveElementsNamespace(name, namespace string) {
	filtered := e.elements[:0]
	for _, elem := range e.elements {
		if elem.Name() != name || elem.Attribute("xmlns") != namespace {
			filtered = append(filtered, elem)
		}
	}
	e.elements = filtered
}

// ClearElements removes all elements.
func (e *XElement) ClearElements() {
	e.elements = nil
}

// IsError returns true if element has a 'type' attribute of value 'error'.
func (e *XElement) IsError() bool {
	return e.Type() == "error"
}

// String returns a string representation of the element.
func (e *XElement) String() string {
	return e.XML(true)
}

// XML satisfies Serializable interface.
func (e *XElement) XML(includeClosing bool) string {
	buf := serializeBufs.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		serializeBufs.Put(buf)
	}()

	buf.WriteString("<")
	buf.WriteString(e.name)

	// serialize attributes
	for i := 0; i < len(e.attrs); i++ {
		if len(e.attrs[i].Value) == 0 {
			continue
		}
		buf.WriteString(" ")
		buf.WriteString(e.attrs[i].Label)
		buf.WriteString(`="`)
		buf.WriteString(e.attrs[i].Value)
		buf.WriteString(`"`)
	}
	textLen := e.TextLen()
	if len(e.elements) > 0 || textLen > 0 {
		buf.WriteString(">")

		// serialize text
		if textLen > 0 {
			buf.WriteString(e.text)
		}
		// serialize child elements
		for j := 0; j < len(e.elements); j++ {
			buf.WriteString(e.elements[j].XML(true))
		}
		if includeClosing {
			buf.WriteString("</")
			buf.WriteString(e.name)
			buf.WriteString(">")
		}
	} else {
		if includeClosing {
			buf.WriteString("/>")
		} else {
			buf.WriteString(">")
		}
	}
	return buf.String()
}
