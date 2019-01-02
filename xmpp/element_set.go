/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

import (
	"encoding/gob"
)

// ElementSet interface represents a read-only set of XML sub elements.
type ElementSet interface {
	// Children returns all elements identified by name.
	// Returns an empty array if no elements are found.
	Children(name string) []XElement

	// Child returns first element identified by name.
	// Returns nil if no element is found.
	Child(name string) XElement

	// ChildrenNamespace returns all elements identified by name and namespace.
	// Returns an empty array if no elements are found.
	ChildrenNamespace(name, namespace string) []XElement

	// ChildNamespace returns first element identified by name and namespace.
	// Returns nil if no element is found.
	ChildNamespace(name, namespace string) XElement

	// All returns a list of all child nodes.
	All() []XElement

	// Count returns child elements count.
	Count() int
}

type elementSet []XElement

func (es elementSet) Children(name string) []XElement {
	var ret []XElement
	for _, node := range es {
		if node.Name() == name {
			ret = append(ret, node)
		}
	}
	return ret
}

func (es elementSet) Child(name string) XElement {
	for _, node := range es {
		if node.Name() == name {
			return node
		}
	}
	return nil
}

func (es elementSet) ChildrenNamespace(name string, namespace string) []XElement {
	var ret []XElement
	for _, node := range es {
		if node.Name() == name && node.Namespace() == namespace {
			ret = append(ret, node)
		}
	}
	return ret
}

func (es elementSet) ChildNamespace(name string, namespace string) XElement {
	for _, node := range es {
		if node.Name() == name && node.Namespace() == namespace {
			return node
		}
	}
	return nil
}

func (es elementSet) All() []XElement {
	return es
}

func (es elementSet) Count() int {
	return len(es)
}

func (es *elementSet) append(nodes ...XElement) {
	*es = append(*es, nodes...)
}

func (es *elementSet) remove(name string) {
	filtered := (*es)[:0]
	for _, node := range *es {
		if node.Name() != name {
			filtered = append(filtered, node)
		}
	}
	*es = filtered
}

func (es *elementSet) removeNamespace(name string, namespace string) {
	filtered := (*es)[:0]
	for _, elem := range *es {
		if elem.Name() != name || elem.Attributes().Get("xmlns") != namespace {
			filtered = append(filtered, elem)
		}
	}
	*es = filtered
}

func (es *elementSet) clear() {
	*es = nil
}

func (es *elementSet) copyFrom(from elementSet) {
	set := make([]XElement, from.Count())
	for i := 0; i < len(from); i++ {
		set[i] = NewElementFromElement(from[i])
	}
	*es = set
}

func (es *elementSet) fromGob(dec *gob.Decoder) {
	var c int
	dec.Decode(&c)
	set := make([]XElement, c)
	for i := 0; i < c; i++ {
		set[i] = NewElementFromGob(dec)
	}
	*es = set
}

func (es elementSet) toGob(enc *gob.Encoder) {
	enc.Encode(len(es))
	for _, el := range es {
		el.ToGob(enc)
	}
}
