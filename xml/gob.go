/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"encoding/gob"
	"io"
)

// FromBytes deserializes XML element from it's gob
// binary representation.
func (e *xElement) FromBytes(r io.Reader) {
	dec := gob.NewDecoder(r)
	dec.Decode(&e.name)
	dec.Decode(&e.text)
	var attrc int
	dec.Decode(&attrc)
	for i := 0; i < attrc; i++ {
		var attr Attribute
		dec.Decode(&attr.Label)
		dec.Decode(&attr.Value)
		e.attrs = append(e.attrs, attr)
	}
	var elemc int
	dec.Decode(&elemc)
	for i := 0; i < elemc; i++ {
		el := &xElement{}
		el.FromBytes(r)
		e.elements = append(e.elements, el)
	}
}

// ToBytes serializes XML element to it's gob
// binary representation.
func (e *xElement) ToBytes(w io.Writer) {
	enc := gob.NewEncoder(w)
	enc.Encode(&e.name)
	enc.Encode(&e.text)
	enc.Encode(len(e.attrs))
	for _, attr := range e.attrs {
		enc.Encode(&attr.Label)
		enc.Encode(&attr.Value)
	}
	enc.Encode(len(e.elements))
	for _, elem := range e.elements {
		elem.ToBytes(w)
	}
}
