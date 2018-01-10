/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"unicode/utf8"
)

var strBufs = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// Attribute represents an XML node attribute (label=value).
type Attribute struct {
	Label string
	Value string
}

// Element represents an XML node element.
type Element interface {
	fmt.Stringer

	// Name returns XML node name.
	Name() string

	// Namespace returns 'xmlns' node attribute.
	Namespace() string

	// ID returns 'id' node attribute.
	ID() string

	// Language returns 'xml:lang' node attribute.
	Language() string

	// Version returns 'version' node attribute.
	Version() string

	// From returns 'from' node attribute.
	From() string

	// To returns 'to' node attribute.
	To() string

	// Type returns 'type' node attribute.
	Type() string

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

	// ToXML serializes element to a raw XML representation.
	// includeClosing determines if closing tag should be attached.
	ToXML(writer io.Writer, includeClosing bool)
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

func (e *XElement) Namespace() string {
	return e.Attribute("xmlns")
}

// SetNamespace sets 'xmlns' node attribute.
func (e *XElement) SetNamespace(namespace string) {
	e.SetAttribute("xmlns", namespace)
}

func (e *XElement) ID() string {
	return e.Attribute("id")
}

// SetID sets 'id' node attribute.
func (e *XElement) SetID(identifier string) {
	e.SetAttribute("id", identifier)
}

func (e *XElement) Language() string {
	return e.Attribute("xml:lang")
}

// SetLanguage sets 'xml:lang' node attribute.
func (e *XElement) SetLanguage(language string) {
	e.SetAttribute("xml:lang", language)
}

func (e *XElement) Version() string {
	return e.Attribute("version")
}

// SetVersion sets 'version' node attribute.
func (e *XElement) SetVersion(version string) {
	e.SetAttribute("version", version)
}

func (e *XElement) From() string {
	return e.Attribute("from")
}

// SetFrom sets 'from' node attribute.
func (e *XElement) SetFrom(from string) {
	e.SetAttribute("from", from)
}

func (e *XElement) To() string {
	return e.Attribute("to")
}

// SetTo sets 'to' node attribute.
func (e *XElement) SetTo(to string) {
	e.SetAttribute("to", to)
}

func (e *XElement) Type() string {
	return e.Attribute("type")
}

// SetType sets 'type' node attribute.
func (e *XElement) SetType(tp string) {
	e.SetAttribute("type", tp)
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
	buf := strBufs.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		strBufs.Put(buf)
	}()
	e.ToXML(buf, true)
	return buf.String()
}

func (e *XElement) ToXML(w io.Writer, includeClosing bool) {
	w.Write([]byte("<"))
	w.Write([]byte(e.name))

	// serialize attributes
	for i := 0; i < len(e.attrs); i++ {
		if len(e.attrs[i].Value) == 0 {
			continue
		}
		w.Write([]byte(" "))
		w.Write([]byte(e.attrs[i].Label))
		w.Write([]byte(`="`))
		w.Write([]byte(e.attrs[i].Value))
		w.Write([]byte(`"`))
	}
	textLen := e.TextLen()
	if len(e.elements) > 0 || textLen > 0 {
		w.Write([]byte(">"))

		// serialize text
		if textLen > 0 {
			w.Write([]byte(e.text))
		}
		// serialize child elements
		for j := 0; j < len(e.elements); j++ {
			e.elements[j].ToXML(w, true)
		}
		if includeClosing {
			w.Write([]byte("</"))
			w.Write([]byte(e.name))
			w.Write([]byte(">"))
		}
	} else {
		if includeClosing {
			w.Write([]byte("/>"))
		} else {
			w.Write([]byte(">"))
		}
	}
}
