/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"encoding/gob"
	"io"
)

type XElement struct {
	name     string
	text     string
	attrs    attributeSet
	elements elementSet
}

// NewElementName creates a mutable XML Element instance with a given name.
func NewElementName(name string) *XElement {
	return &XElement{name: name}
}

// NewElementNamespace creates a mutable XML Element instance with a given name and namespace.
func NewElementNamespace(name, namespace string) *XElement {
	return &XElement{
		name:  name,
		attrs: attributeSet{attrs: []Attribute{{"xmlns", namespace}}},
	}
}

// NewElementFromElement creates a mutable XML Element by copying an element.
func NewElementFromElement(elem Element) *XElement {
	e := &XElement{}
	e.copyFrom(elem)
	return e
}

// NewElementFromGob deserializes an element node from it's gob binary representation.
func NewElementFromGob(dec *gob.Decoder) *XElement {
	e := &XElement{}
	dec.Decode(&e.name)
	dec.Decode(&e.text)
	e.attrs.fromGob(dec)
	e.elements.fromGob(dec)
	return e
}

// NewErrorElementFromElement returns a copy of an element of stanza error class.
func NewErrorElementFromElement(elem Element, stanzaErr *StanzaError) *XElement {
	e := &XElement{}
	e.copyFrom(elem)
	e.SetAttribute("type", "error")
	e.AppendElement(stanzaErr.Element())
	return e
}

// Name returns XML node name.
func (e *XElement) Name() string {
	return e.name
}

// Attributes returns XML node attribute value.
func (e *XElement) Attributes() AttributeSet {
	return &e.attrs
}

// Elements returns all instance's child elements.
func (e *XElement) Elements() ElementSet {
	return &e.elements
}

// Text returns XML node text value.
// Returns an empty string if not set.
func (e *XElement) Text() string {
	return e.text
}

// Namespace returns 'xmlns' node attribute.
func (e *XElement) Namespace() string {
	return e.attrs.Get("xmlns")
}

// ID returns 'id' node attribute.
func (e *XElement) ID() string {
	return e.attrs.Get("id")
}

// Language returns 'xml:lang' node attribute.
func (e *XElement) Language() string {
	return e.attrs.Get("xml:lang")
}

// SetVersion sets 'version' node attribute.
func (e *XElement) SetVersion(version string) {
	e.attrs.setAttribute("version", version)
}

// Version returns 'version' node attribute.
func (e *XElement) Version() string {
	return e.attrs.Get("version")
}

// From returns 'from' node attribute.
func (e *XElement) From() string {
	return e.attrs.Get("from")
}

// To returns 'to' node attribute.
func (e *XElement) To() string {
	return e.attrs.Get("to")
}

// Type returns 'type' node attribute.
func (e *XElement) Type() string {
	return e.attrs.Get("type")
}

// IsError returns true if element has a 'type' attribute of value 'error'.
func (e *XElement) IsError() bool {
	return e.Type() == ErrorType
}

// Error returns element error sub element.
func (e *XElement) Error() Element {
	return e.elements.Child("error")
}

// String returns a string representation of the element.
func (e *XElement) String() string {
	buf := pool.Get()
	defer pool.Put(buf)

	e.ToXML(buf, true)
	return buf.String()
}

// ToXML serializes element to a raw XML representation.
// includeClosing determines if closing tag should be attached.
func (e *XElement) ToXML(w io.Writer, includeClosing bool) {
	w.Write([]byte("<"))
	w.Write([]byte(e.name))

	// serialize attributes
	e.attrs.toXML(w)

	textLen := len(e.text)
	if e.elements.Count() > 0 || textLen > 0 {
		w.Write([]byte(">"))

		// serialize text
		if textLen > 0 {
			escapeText(w, []byte(e.text), false)
		}
		// serialize child elements
		e.elements.toXML(w)

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

// ToGob serializes an element node to it's gob binary representation.
func (e *XElement) ToGob(enc *gob.Encoder) {
	enc.Encode(&e.name)
	enc.Encode(&e.text)
	e.attrs.toGob(enc)
	e.elements.toGob(enc)
}

func (e *XElement) copyFrom(el Element) {
	e.name = el.Name()
	e.text = el.Text()
	e.attrs.copyFrom(el.Attributes().(*attributeSet))
	e.elements.copyFrom(el.Elements().(*elementSet))
}
