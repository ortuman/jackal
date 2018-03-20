/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/ortuman/jackal/bufferpool"
)

// ErrorType represents an 'error' stanza type.
const ErrorType = "error"

var pool = bufferpool.New()

// Attribute represents an XML node attribute (label=value).
type Attribute struct {
	Label string
	Value string
}

// Element represents an XML node element.
type Element interface {
	fmt.Stringer

	Name() string
	Namespace() string

	ID() string
	Language() string
	Version() string
	From() string
	To() string
	Type() string

	Text() string
	TextLen() int

	Attribute(label string) string
	Attributes() []Attribute
	AttributesCount() int

	FindElement(name string) Element
	FindElements(name string) []Element
	FindElementNamespace(name, namespace string) Element
	FindElementsNamespace(name, namespace string) []Element

	Elements() []Element
	ElementsCount() int

	ToError(stanzaError *StanzaError) Element
	Error() Element

	ToXML(writer io.Writer, includeClosing bool)

	FromBytes(r io.Reader)
	ToBytes(w io.Writer)
}

type xElement struct {
	name     string
	text     string
	attrs    []Attribute
	elements []Element
}

// Name returns XML node name.
func (e *xElement) Name() string {
	return e.name
}

// Namespace returns 'xmlns' node attribute.
func (e *xElement) Namespace() string {
	return e.Attribute("xmlns")
}

// ID returns 'id' node attribute.
func (e *xElement) ID() string {
	return e.Attribute("id")
}

// Language returns 'xml:lang' node attribute.
func (e *xElement) Language() string {
	return e.Attribute("xml:lang")
}

// Version returns 'version' node attribute.
func (e *xElement) Version() string {
	return e.Attribute("version")
}

// From returns 'from' node attribute.
func (e *xElement) From() string {
	return e.Attribute("from")
}

// To returns 'to' node attribute.
func (e *xElement) To() string {
	return e.Attribute("to")
}

// Type returns 'type' node attribute.
func (e *xElement) Type() string {
	return e.Attribute("type")
}

// Text returns XML node text value.
// Returns an empty string if not set.
func (e *xElement) Text() string {
	return e.text
}

// TextLen returns XML node text value length.
func (e *xElement) TextLen() int {
	return utf8.RuneCountInString(e.text)
}

// Attribute returns XML node attribute value.
func (e *xElement) Attribute(label string) string {
	for _, attr := range e.attrs {
		if attr.Label == label {
			return attr.Value
		}
	}
	return ""
}

// Attributes returns all XML node attributes.
func (e *xElement) Attributes() []Attribute {
	return e.attrs
}

// AttributesCount XML attributes count.
func (e *xElement) AttributesCount() int {
	return len(e.attrs)
}

// FindElement returns first element identified by name.
// Returns nil if no element is found.
func (e *xElement) FindElement(name string) Element {
	for _, element := range e.elements {
		if element.Name() == name {
			return element
		}
	}
	return nil
}

// FindElements returns all elements identified by name.
// Returns an empty array if no elements are found.
func (e *xElement) FindElements(name string) []Element {
	var ret []Element
	for _, element := range e.elements {
		if element.Name() == name {
			ret = append(ret, element)
		}
	}
	return ret
}

// FindElementNamespace returns first element identified by name and namespace.
// Returns nil if no element is found.
func (e *xElement) FindElementNamespace(name, namespace string) Element {
	for _, element := range e.elements {
		if element.Name() == name && element.Namespace() == namespace {
			return element
		}
	}
	return nil
}

// FindElementsNamespace returns all elements identified by name and namespace.
// Returns an empty array if no elements are found.
func (e *xElement) FindElementsNamespace(name, namespace string) []Element {
	var ret []Element
	for _, element := range e.elements {
		if element.Name() == name && element.Namespace() == namespace {
			ret = append(ret, element)
		}
	}
	return ret
}

// Elements returns all instance's child elements.
func (e *xElement) Elements() []Element {
	return e.elements
}

// ElementsCount returns child elements count.
func (e *xElement) ElementsCount() int {
	return len(e.elements)
}

// IsError returns true if element has a 'type' attribute of value 'error'.
func (e *xElement) IsError() bool {
	return e.Type() == ErrorType
}

// Error returns element error sub element.
func (e *xElement) Error() Element {
	return e.FindElement("error")
}

// String returns a string representation of the element.
func (e *xElement) String() string {
	buf := pool.Get()
	defer pool.Put(buf)

	e.ToXML(buf, true)
	return buf.String()
}

// ToXML serializes element to a raw XML representation.
// includeClosing determines if closing tag should be attached.
func (e *xElement) ToXML(w io.Writer, includeClosing bool) {
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
			escapeText(w, []byte(e.text), false)
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

// Copy returns a deep copy of this message stanza.
func (e *xElement) Copy() *MutableElement {
	cp := &MutableElement{}
	cp.copyFrom(e)
	return cp
}

func (e *xElement) copyFrom(el Element) {
	e.name = el.Name()
	e.text = el.Text()
	e.attrs = make([]Attribute, el.AttributesCount())
	copy(e.attrs, el.Attributes())

	els := el.Elements()
	e.elements = make([]Element, len(els))
	for i := 0; i < len(els); i++ {
		el := &xElement{}
		el.copyFrom(els[i])
		e.elements[i] = el
	}
}

func (e *xElement) setAttribute(label, value string) {
	for i := 0; i < len(e.attrs); i++ {
		if e.attrs[i].Label == label {
			e.attrs[i].Value = value
			return
		}
	}
	e.attrs = append(e.attrs, Attribute{label, value})
}

func (e *xElement) removeAttribute(label string) {
	for i := 0; i < len(e.attrs); i++ {
		if e.attrs[i].Label == label {
			e.attrs = append(e.attrs[:i], e.attrs[i+1:]...)
			return
		}
	}
}

func (e *xElement) appendElement(element Element) {
	e.elements = append(e.elements, element)
}

func (e *xElement) appendElements(elements []Element) {
	e.elements = append(e.elements, elements...)
}

func (e *xElement) removeElements(name string) {
	filtered := e.elements[:0]
	for _, elem := range e.elements {
		if elem.Name() != name {
			filtered = append(filtered, elem)
		}
	}
	e.elements = filtered
}

func (e *xElement) removeElementsNamespace(name, namespace string) {
	filtered := e.elements[:0]
	for _, elem := range e.elements {
		if elem.Name() != name || elem.Attribute("xmlns") != namespace {
			filtered = append(filtered, elem)
		}
	}
	e.elements = filtered
}
