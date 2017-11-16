/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

// Attribute represents an XML node attribute (label=value).
type Attribute struct {
	label string
	value string
}

// Element represents an XML node element.
type Element struct {
	name   string
	text   string
	attrs  []Attribute
	childs []*Element
}

// NewElement creates an XML Element instance with a given name.
func NewElement(name string) *Element {
	e := Element{}
	e.name = name
	e.attrs = []Attribute{}
	e.childs = []*Element{}
	return &e
}

// NewElementNS creates an XML Element instance with a given name and namespace.
func NewElementNS(name, namespace string) *Element {
	e := Element{}
	e.name = name
	e.attrs = []Attribute{{"xmlns", namespace}}
	e.childs = []*Element{}
	return &e
}

// Name returns XML node name.
func (e *Element) Name() string {
	return e.name
}

// Attribute returns XML node attribute value.
func (e *Element) Attribute(label string) string {
	for i := 0; i < len(e.attrs); i++ {
		if e.attrs[i].label == label {
			return e.attrs[i].value
		}
	}
	return ""
}

// FindElement returns first element identified by name.
// Returns nil if no element is found.
func (e *Element) FindElement(name string) *Element {
	for i := 0; i < len(e.childs); i++ {
		if e.childs[i].name == name {
			return e.childs[i]
		}
	}
	return nil
}

// FindElements returns all elements identified by name.
// Returns an empty array if no elements are found.
func (e *Element) FindElements(name string) []*Element {
	ret := e.childs[:0]
	for i := 0; i < len(e.childs); i++ {
		if e.childs[i].name == name {
			ret = append(ret, e.childs[i])
		}
	}
	return ret
}

// FindElementNS returns first element identified by name and namespace.
// Returns nil if no element is found.
func (e *Element) FindElementNS(name, namespace string) *Element {
	for i := 0; i < len(e.childs); i++ {
		if e.childs[i].name == name && e.childs[i].Namespace() == namespace {
			return e.childs[i]
		}
	}
	return nil
}

// FindElementsNS returns all elements identified by name and namespace.
// Returns an empty array if no elements are found.
func (e *Element) FindElementsNS(name, namespace string) []*Element {
	ret := e.childs[:0]
	for i := 0; i < len(e.childs); i++ {
		if e.childs[i].name == name && e.childs[i].Namespace() == namespace {
			ret = append(ret, e.childs[i])
		}
	}
	return ret
}

// ElementsCount returns child elements count.
func (e *Element) ElementsCount() int {
	return len(e.childs)
}

// Text returns XML node text value.
// Returns an empty string if not set.
func (e *Element) Text() string {
	return e.text
}
