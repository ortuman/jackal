/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import (
	"sync/atomic"
	"unsafe"
)

// An Attribute represents an attribute in an XML element (label=value).
type Attribute struct {
	label string
	value string
}

type Element struct {
	p        unsafe.Pointer
	shadowed int32
}

// NewElement creates an XML Element instance with name and attributes.
func NewElement(name string, attributes []Attribute) *Element {
	shared := newElement(name, attributes)
	e := &Element{}
	e.p = unsafe.Pointer(shared)
	e.shadowed = 0
	return e
}

// NewElementName creates a new Element with a given name.
func NewElementName(name string) *Element {
	return NewElement(name, []Attribute{})
}

// NewElementNamespace creates a new Element with a given name and namespace.
func NewElementNamespace(name, namespace string) *Element {
	return NewElement(name, []Attribute{{"namespace", namespace}})
}

// SetAttribute writes or replace an XML node attribute value.
func (e *Element) SetAttribute(label, value string) {
	e.copyOnWrite()
	e.shared().setAttribute(label, value)
}

// GetAttribute returns XML node attribute value.
func (e *Element) Attribute(label string) string {
	return e.shared().attribute(label)
}

// AppendElement appends an XML node element.
func (e *Element) AppendElement(element Element) {
	e.copyOnWrite()
	e.shared().appendElement(element)
}

// AppendElements appends an array of XML node elements.
func (e *Element) AppendElements(elements []Element) {
	e.copyOnWrite()
	e.shared().appendElements(elements)
}

// GetElement returns first element identified by name.
// Returns nil if no element is found.
func (e *Element) Element(name string) *Element {
	return e.shared().element(name)
}

// GetElementNamespace returns first element identified by name and namespace.
// Returns nil if no element is found.
func (e *Element) ElementNamespace(name, namespace string) *Element {
	return e.shared().elementNamespace(name, namespace)
}

// GetElements returns all elements identified by name.
// Returns nil if no elements are found.
func (e *Element) Elements(name string) []Element {
	return e.shared().elements(name)
}

// GetElementsNamespace returns all elements identified by name and namespace.
// Returns nil if no elements are found.
func (e *Element) ElementsNamespace(name, namespace string) []Element {
	return e.shared().elementsNamespace(name, namespace)
}

// SetText
func (e *Element) SetText(text string) {
	e.copyOnWrite()
	e.shared().text = text
}

// GetText returns XML node text value.
// Returns empty string if not set.
func (e *Element) Text() string {
	return e.shared().text
}

func (e *Element) shared() *element {
	return (*element)(atomic.LoadPointer(&e.p))
}

func (e *Element) copyOnWrite() {
	if atomic.CompareAndSwapInt32(&e.shadowed, 0, 1) {
		atomic.StorePointer(&e.p, unsafe.Pointer(e.shared().copy()))
	}
}

type element struct {
	name          string
	text          string
	attributes    []Attribute
	childElements []Element
}

func newElement(name string, attributes []Attribute) *element {
	e := &element{}
	e.name = name
	e.attributes = attributes
	e.childElements = []Element{}
	return e
}

func (e *element) setAttribute(label, value string) {
	for i := 0; i < len(e.attributes); i++ {
		if e.attributes[i].label == label {
			e.attributes[i].value = value
			return
		}
	}
	e.attributes = append(e.attributes, Attribute{label, value})
}

func (e *element) attribute(label string) string {
	for i := 0; i < len(e.attributes); i++ {
		if e.attributes[i].label == label {
			return e.attributes[i].value
		}
	}
	return ""
}

func (e *element) appendElement(childElement Element) {
	e.childElements = append(e.childElements, childElement)
}

func (e *element) appendElements(childElements []Element) {
	e.childElements = append(e.childElements, childElements...)
}

func (e *element) element(name string) *Element {
	return e.elementNamespace(name, "")
}

func (e *element) elementNamespace(name, namespace string) *Element {
	for i := 0; i < len(e.childElements); i++ {
		shared := e.childElements[i].shared()
		if shared.name == name && shared.attribute("namespace") == namespace {
			return &e.childElements[i]
		}
	}
	return nil
}

func (e *element) elements(name string) []Element {
	return e.elementsNamespace(name, "")
}

func (e *element) elementsNamespace(name, namespace string) []Element {
	ret := e.childElements[:0]
	for _, c := range e.childElements {
		if c.shared().name == name && c.shared().attribute("namespace") == namespace {
			ret = append(ret, c)
		}
	}
	return ret
}

func (e *element) copy() *element {
	cp := &element{}
	cp.name = e.name
	cp.attributes = make([]Attribute, len(e.attributes), cap(e.attributes))
	cp.childElements = make([]Element, len(e.childElements), cap(e.childElements))
	copy(cp.attributes, e.attributes)
	copy(cp.childElements, e.childElements)
	return cp
}
