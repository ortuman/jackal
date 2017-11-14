/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import (
	"sync/atomic"
	"unsafe"
)

type Attribute struct {
	label string
	value string
}

type element struct {
	name          string
	attributes    []Attribute
	childElements []Element
	text          string
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

func (e *element) appendElement(childElement Element) {
	e.childElements = append(e.childElements, childElement)
}

func (e *element) getElement(name string) *Element {
	for i := 0; i < len(e.childElements); i++ {
		if e.childElements[i].shared().name == name {
			return &e.childElements[i]
		}
	}
	return nil
}

func (e *element) copy() *element {
	cp := &element{}
	cp.name = e.name
	cp.attributes = make([]Attribute, len(e.attributes), cap(e.attributes))
	cp.childElements = make([]Element, len(e.childElements), cap(e.childElements))
	return cp
}

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

func NewElementName(name string) *Element {
	return NewElement(name, []Attribute{})
}

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

func (e *Element) GetElement(name string) *Element {
	return e.shared().getElement(name)
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
	if atomic.LoadInt32(&e.shadowed) > 0 {
		return
	}
	atomic.StorePointer(&e.p, unsafe.Pointer(e.shared().copy()))
	atomic.StoreInt32(&e.shadowed, 1)
}
