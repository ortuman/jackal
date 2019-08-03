/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

import (
	"bytes"
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

func (es *elementSet) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	var c int
	if err := dec.Decode(&c); err != nil {
		return err
	}
	if c > 0 {
		set := make([]XElement, c)
		for i := 0; i < c; i++ {
			el, err := NewElementFromBytes(buf)
			if err != nil {
				return err
			}
			set[i] = el
		}
		*es = set
	}
	return nil
}

func (es elementSet) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(len(es)); err != nil {
		return err
	}
	for _, el := range es {
		if err := el.ToBytes(buf); err != nil {
			return err
		}
	}
	return nil
}
