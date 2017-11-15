/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import (
	"sync/atomic"
	"unsafe"
)

// Attribute represents an XML node attribute (label=value).
type Attribute struct {
	label string
	value string
}

// Element represents an XML node element.
type Element struct {
	p        unsafe.Pointer
	shadowed int32
}

// NewElement creates an XML Element instance with name and attributes.
func NewElement(name string, attributes []Attribute) Element {
	shared := newElement(name, attributes)
	e := Element{}
	e.p = unsafe.Pointer(shared)
	e.shadowed = 0
	return e
}

// NewElementName creates a new Element with a given name.
func NewElementName(name string) Element {
	return NewElement(name, []Attribute{})
}

// NewElementNamespace creates a new Element with a given name and namespace.
func NewElementNamespace(name, namespace string) Element {
	return NewElement(name, []Attribute{{"xmlns", namespace}})
}

// SetName sets XML node name.
func (e *Element) SetName(name string) {
	e.copyOnWrite()
	e.shared().name = name
}

// Name returns XML node name.
func (e *Element) Name() string {
	return e.shared().name
}

// SetAttribute sets an XML node attribute value.
func (e *Element) SetAttribute(label, value string) {
	e.copyOnWrite()
	e.shared().setAttribute(label, value)
}

// RemoveAttribute removes an XML node attribute.
func (e *Element) RemoveAttribute(label string) {
	e.copyOnWrite()
	e.shared().removeAttribute(label)
}

// Attribute returns XML node attribute value.
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

// RemoveElements removes elements identified by name.
func (e *Element) RemoveElements(name string) {
	e.copyOnWrite()
	e.shared().removeElements(name)
}

// RemoveElementsNamespace removes elements identified by name and namespace.
func (e *Element) RemoveElementsNamespace(name, namespace string) {
	e.copyOnWrite()
	e.shared().removeElementsNamespace(name, namespace)
}

// Element returns first element identified by name.
// Returns nil if no element is found.
func (e *Element) Element(name string) *Element {
	return e.shared().element(name)
}

// ElementNamespace returns first element identified by name and namespace.
// Returns nil if no element is found.
func (e *Element) ElementNamespace(name, namespace string) *Element {
	return e.shared().elementNamespace(name, namespace)
}

// Elements returns all elements identified by name.
// Returns nil if no elements are found.
func (e *Element) Elements(name string) []Element {
	return e.shared().elements(name)
}

// ElementsNamespace returns all elements identified by name and namespace.
// Returns nil if no elements are found.
func (e *Element) ElementsNamespace(name, namespace string) []Element {
	return e.shared().elementsNamespace(name, namespace)
}

// SetText sets XML node text value.
func (e *Element) SetText(text string) {
	e.copyOnWrite()
	e.shared().text = text
}

// Text returns XML node text value.
// Returns an empty string if not set.
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

func (e *element) removeAttribute(label string) {
	j := -1
	for i := 0; i < len(e.attributes); i++ {
		if e.attributes[i].label == label {
			j = i
			break
		}
	}
	if j != -1 {
		e.attributes = append(e.attributes[:j], e.attributes[j+1:]...)
	}
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

func (e *element) removeElements(name string) {
	e.removeElementsNamespace(name, "")
}

func (e *element) removeElementsNamespace(name, namespace string) {
	childElements := e.childElements[:0]
	for _, c := range e.childElements {
		matches := c.Name() == name && c.Attribute("xmlns") == namespace
		if !matches {
			childElements = append(childElements, c)
		}
	}
	e.childElements = childElements
}

func (e *element) element(name string) *Element {
	return e.elementNamespace(name, "")
}

func (e *element) elementNamespace(name, namespace string) *Element {
	for i := 0; i < len(e.childElements); i++ {
		sh := e.childElements[i].shared()
		if sh.name == name && sh.attribute("xmlns") == namespace {
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
		if c.Name() == name && c.Attribute("xmlns") == namespace {
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
