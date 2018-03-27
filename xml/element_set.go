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
	Children(name string) []ElementNode

	// Child returns first element identified by name.
	// Returns nil if no element is found.
	Child(name string) ElementNode

	// ChildrenNamespace returns all elements identified by name and namespace.
	// Returns an empty array if no elements are found.
	ChildrenNamespace(name, namespace string) []ElementNode

	// ChildNamespace returns first element identified by name and namespace.
	// Returns nil if no element is found.
	ChildNamespace(name, namespace string) ElementNode

	// All returns a list of all child nodes.
	All() []ElementNode

	// Count returns child elements count.
	Count() int
}

type elementSet struct {
	nodes []ElementNode
}

func (es *elementSet) Children(name string) []ElementNode {
	var ret []ElementNode
	for _, node := range es.nodes {
		if node.Name() == name {
			ret = append(ret, node)
		}
	}
	return ret
}

func (es *elementSet) Child(name string) ElementNode {
	for _, node := range es.nodes {
		if node.Name() == name {
			return node
		}
	}
	return nil
}

func (es *elementSet) ChildrenNamespace(name string, namespace string) []ElementNode {
	var ret []ElementNode
	for _, node := range es.nodes {
		if node.Name() == name && node.Namespace() == namespace {
			ret = append(ret, node)
		}
	}
	return ret
}

func (es *elementSet) ChildNamespace(name string, namespace string) ElementNode {
	for _, node := range es.nodes {
		if node.Name() == name && node.Namespace() == namespace {
			return node
		}
	}
	return nil
}

func (es *elementSet) All() []ElementNode {
	return es.nodes
}

func (es *elementSet) Count() int {
	return len(es.nodes)
}

func (es *elementSet) append(nodes ...ElementNode) {
	es.nodes = append(es.nodes, nodes...)
}

func (es *elementSet) remove(name string) {
	filtered := es.nodes[:0]
	for _, node := range es.nodes {
		if node.Name() != name {
			filtered = append(filtered, node)
		}
	}
	es.nodes = filtered
}

func (es *elementSet) removeNamespace(name string, namespace string) {
	filtered := es.nodes[:0]
	for _, elem := range es.nodes {
		if elem.Name() != name || elem.Attributes().Get("xmlns") != namespace {
			filtered = append(filtered, elem)
		}
	}
	es.nodes = filtered
}

func (es *elementSet) clear() {
	es.nodes = nil
}

func (es *elementSet) copyFrom(from *elementSet) {
	es.nodes = make([]ElementNode, from.Count())
	copy(es.nodes, from.nodes)
}

func (es *elementSet) toXML(w io.Writer) {
	for j := 0; j < len(es.nodes); j++ {
		es.nodes[j].ToXML(w, true)
	}
}

func (es *elementSet) fromGob(dec *gob.Decoder) {
	var c int
	dec.Decode(&c)
	for i := 0; i < c; i++ {
		es.nodes = append(es.nodes, NewElementFromGob(dec))
	}
}

func (es *elementSet) toGob(enc *gob.Encoder) {
	enc.Encode(len(es.nodes))
	for _, node := range es.nodes {
		node.ToGob(enc)
	}
}
