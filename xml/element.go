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

// element represents details for the XML representation node.
type element struct {
	name          string
	text          string
	attributes    []Attribute
	childElements []Element
}

func newElement(name string, attributes []Attribute, childElements []Element) *element {
	e := &element{}
	e.name = name
	e.attributes = attributes
	e.childElements = childElements
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

func (e *element) getAttribute(label string) string {
	for i := 0; i < len(e.attributes); i++ {
		if e.attributes[i].label == label {
			return e.attributes[i].value
		}
	}
	return ""
}

// Appends a child element.
func (e *element) appendElement(childElement Element) {
	e.childElements = append(e.childElements, childElement)
}

// Appends an array of child elements.
func (e *element) appendElements(childElements []Element) {
	e.childElements = append(e.childElements, childElements...)
}

// Returns element identified by name, or nil if not found.
func (e *element) getElement(name string) *Element {
	return e.getElementNamespace(name, "")
}

// Returns element identified by name and namespace, or nil if not found.
func (e *element) getElementNamespace(name, namespace string) *Element {
	for i := 0; i < len(e.childElements); i++ {
		shared := e.childElements[i].shared()
		if shared.name == name && shared.getAttribute("namespace") == namespace {
			return &e.childElements[i]
		}
	}
	return nil
}

func (e *element) getElements(name string) []Element {
	return e.getElementsNamespace(name, "")
}

func (e *element) getElementsNamespace(name, namespace string) []Element {
	ret := e.childElements[:0]
	for _, c := range e.childElements {
		if c.shared().name == name && c.shared().getAttribute("namespace") == namespace {
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

// Element represents a shadowed copy of an XML
type Element struct {
	p        unsafe.Pointer
	shadowed int32
}

func NewElement(name string, attributes []Attribute) *Element {
	shared := newElement(name, attributes, []Element{})
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

func (e *Element) SetAttribute(label, value string) {
	e.copyOnWrite()
	e.shared().setAttribute(label, value)
}

func (e *Element) GetAttribute(label string) string {
	return e.shared().getAttribute(label)
}

func (e *Element) AppendElement(element Element) {
	e.copyOnWrite()
	e.shared().appendElement(element)
}

func (e *Element) AppendElements(elements []Element) {
	e.copyOnWrite()
	e.shared().appendElements(elements)
}

func (e *Element) GetElement(name string) *Element {
	return e.shared().getElement(name)
}

func (e *Element) GetElementNamespace(name, namespace string) *Element {
	return e.shared().getElementNamespace(name, namespace)
}

func (e *Element) GetElements(name string) []Element {
	return e.shared().getElements(name)
}

func (e *Element) GetElementsNamespace(name, namespace string) []Element {
	return e.shared().getElementsNamespace(name, namespace)
}

func (e *Element) SetText(text string) {
	e.copyOnWrite()
	e.shared().text = text
}

func (e *Element) GetText() string {
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
