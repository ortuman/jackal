/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"encoding/gob"
	"io"
)

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

type elementSet struct {
	elems []XElement
}

func (es *elementSet) Children(name string) []XElement {
	var ret []XElement
	for _, node := range es.elems {
		if node.Name() == name {
			ret = append(ret, node)
		}
	}
	return ret
}

func (es *elementSet) Child(name string) XElement {
	for _, node := range es.elems {
		if node.Name() == name {
			return node
		}
	}
	return nil
}

func (es *elementSet) ChildrenNamespace(name string, namespace string) []XElement {
	var ret []XElement
	for _, node := range es.elems {
		if node.Name() == name && node.Namespace() == namespace {
			ret = append(ret, node)
		}
	}
	return ret
}

func (es *elementSet) ChildNamespace(name string, namespace string) XElement {
	for _, node := range es.elems {
		if node.Name() == name && node.Namespace() == namespace {
			return node
		}
	}
	return nil
}

func (es *elementSet) All() []XElement {
	return es.elems
}

func (es *elementSet) Count() int {
	return len(es.elems)
}

func (es *elementSet) String() string {
	buf := bufPool.Get()
	defer bufPool.Put(buf)
	es.toXML(buf)
	return buf.String()
}

func (es *elementSet) append(nodes ...XElement) {
	es.elems = append(es.elems, nodes...)
}

func (es *elementSet) remove(name string) {
	filtered := es.elems[:0]
	for _, node := range es.elems {
		if node.Name() != name {
			filtered = append(filtered, node)
		}
	}
	es.elems = filtered
}

func (es *elementSet) removeNamespace(name string, namespace string) {
	filtered := es.elems[:0]
	for _, elem := range es.elems {
		if elem.Name() != name || elem.Attributes().Get("xmlns") != namespace {
			filtered = append(filtered, elem)
		}
	}
	es.elems = filtered
}

func (es *elementSet) clear() {
	es.elems = nil
}

func (es *elementSet) copyFrom(from *elementSet) {
	es.elems = make([]XElement, from.Count())
	for i := 0; i < len(from.elems); i++ {
		es.elems[i] = NewElementFromElement(from.elems[i])
	}
}

func (es *elementSet) toXML(w io.Writer) {
	for j := 0; j < len(es.elems); j++ {
		es.elems[j].ToXML(w, true)
	}
}

func (es *elementSet) fromGob(dec *gob.Decoder) {
	var c int
	dec.Decode(&c)
	es.elems = make([]XElement, c)
	for i := 0; i < c; i++ {
		es.elems[i] = NewElementFromGob(dec)
	}
}

func (es *elementSet) toGob(enc *gob.Encoder) {
	enc.Encode(len(es.elems))
	for _, el := range es.elems {
		el.ToGob(enc)
	}
}
