/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"encoding/gob"
	"io"
)

type Element struct {
	name     string
	text     string
	attrs    attributeSet
	elements elementSet
}

// NewElementName creates a mutable XML XElement instance with a given name.
func NewElementName(name string) *Element {
	return &Element{name: name}
}

// NewElementNamespace creates a mutable XML XElement instance with a given name and namespace.
func NewElementNamespace(name, namespace string) *Element {
	return &Element{
		name:  name,
		attrs: attributeSet{attrs: []Attribute{{"xmlns", namespace}}},
	}
}

// NewElementFromElement creates a mutable XML XElement by copying an element.
func NewElementFromElement(elem XElement) *Element {
	e := &Element{}
	e.copyFrom(elem)
	return e
}

// NewErrorElementFromElement returns a copy of an element of stanza error class.
func NewErrorElementFromElement(elem XElement, stanzaErr *StanzaError, children []XElement) *Element {
	e := &Element{}
	e.copyFrom(elem)
	e.SetType("error")
	e.SetFrom(elem.To())
	e.SetTo(elem.From())
	e.AppendElement(stanzaErr.Element())
	e.AppendElements(children)
	return e
}

// Name returns XML node name.
func (e *Element) Name() string {
	return e.name
}

// Attributes returns XML node attribute value.
func (e *Element) Attributes() AttributeSet {
	return &e.attrs
}

// Elements returns all instance's child elements.
func (e *Element) Elements() ElementSet {
	return &e.elements
}

// Text returns XML node text value.
// Returns an empty string if not set.
func (e *Element) Text() string {
	return e.text
}

// Namespace returns 'xmlns' node attribute.
func (e *Element) Namespace() string {
	return e.attrs.Get("xmlns")
}

// ID returns 'id' node attribute.
func (e *Element) ID() string {
	return e.attrs.Get("id")
}

// Language returns 'xml:lang' node attribute.
func (e *Element) Language() string {
	return e.attrs.Get("xml:lang")
}

// SetVersion sets 'version' node attribute.
func (e *Element) SetVersion(version string) {
	e.attrs.setAttribute("version", version)
}

// Version returns 'version' node attribute.
func (e *Element) Version() string {
	return e.attrs.Get("version")
}

// From returns 'from' node attribute.
func (e *Element) From() string {
	return e.attrs.Get("from")
}

// To returns 'to' node attribute.
func (e *Element) To() string {
	return e.attrs.Get("to")
}

// Type returns 'type' node attribute.
func (e *Element) Type() string {
	return e.attrs.Get("type")
}

// IsError returns true if element has a 'type' attribute of value 'error'.
func (e *Element) IsError() bool {
	return e.Type() == ErrorType
}

// Error returns element error sub element.
func (e *Element) Error() XElement {
	return e.elements.Child("error")
}

// String returns a string representation of the element.
func (e *Element) String() string {
	buf := bufPool.Get()
	defer bufPool.Put(buf)

	e.ToXML(buf, true)
	return buf.String()
}

// ToXML serializes element to a raw XML representation.
// includeClosing determines if closing tag should be attached.
func (e *Element) ToXML(w io.Writer, includeClosing bool) {
	io.WriteString(w, "<")
	io.WriteString(w, e.name)

	// serialize attributes
	for _, attr := range e.attrs.attrs {
		if len(attr.Value) == 0 {
			continue
		}
		io.WriteString(w, " ")
		io.WriteString(w, attr.Label)
		io.WriteString(w, `="`)
		io.WriteString(w, attr.Value)
		io.WriteString(w, `"`)
	}

	if e.elements.Count() > 0 || len(e.text) > 0 {
		io.WriteString(w, ">")

		if len(e.text) > 0 {
			escapeText(w, []byte(e.text), false)
		}
		for _, elem := range e.elements.elems {
			elem.ToXML(w, true)
		}

		if includeClosing {
			io.WriteString(w, "</")
			io.WriteString(w, e.name)
			io.WriteString(w, ">")
		}
	} else {
		if includeClosing {
			io.WriteString(w, "/>")
		} else {
			io.WriteString(w, ">")
		}
	}
}

// FromGob deserializes an element node from it's gob binary representation.
func (e *Element) FromGob(dec *gob.Decoder) {
	dec.Decode(&e.name)
	dec.Decode(&e.text)
	e.attrs.fromGob(dec)
	e.elements.fromGob(dec)
}

// ToGob serializes an element node to it's gob binary representation.
func (e *Element) ToGob(enc *gob.Encoder) {
	enc.Encode(&e.name)
	enc.Encode(&e.text)
	e.attrs.toGob(enc)
	e.elements.toGob(enc)
}

func (e *Element) copyFrom(el XElement) {
	e.name = el.Name()
	e.text = el.Text()
	e.attrs.copyFrom(el.Attributes().(*attributeSet))
	e.elements.copyFrom(el.Elements().(*elementSet))
}
