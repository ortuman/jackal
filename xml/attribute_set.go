/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"encoding/gob"
)

// Attribute represents an XML node attribute (label=value).
type Attribute struct {
	Label string
	Value string
}

type AttributeSet interface {
	Get(string) string
	Count() int
}

type attributeSet struct {
	attrs []Attribute
}

func (as *attributeSet) Get(label string) string {
	for _, attr := range as.attrs {
		if attr.Label == label {
			return attr.Value
		}
	}
	return ""
}

func (as *attributeSet) Count() int {
	return len(as.attrs)
}

func (as *attributeSet) setAttribute(label, value string) {
	for i := 0; i < len(as.attrs); i++ {
		if as.attrs[i].Label == label {
			as.attrs[i].Value = value
			return
		}
	}
	as.attrs = append(as.attrs, Attribute{label, value})
}

func (as *attributeSet) removeAttribute(label string) {
	for i := 0; i < len(as.attrs); i++ {
		if as.attrs[i].Label == label {
			as.attrs = append(as.attrs[:i], as.attrs[i+1:]...)
			return
		}
	}
}

func (as *attributeSet) copyFrom(from *attributeSet) {
	as.attrs = make([]Attribute, from.Count())
	copy(as.attrs, from.attrs)
}

func (as *attributeSet) fromGob(dec *gob.Decoder) {
	var c int
	dec.Decode(&c)
	for i := 0; i < c; i++ {
		var attr Attribute
		dec.Decode(&attr.Label)
		dec.Decode(&attr.Value)
		as.attrs = append(as.attrs, attr)
	}
}

func (as *attributeSet) toGob(enc *gob.Encoder) {
	enc.Encode(len(as.attrs))
	for _, attr := range as.attrs {
		enc.Encode(&attr.Label)
		enc.Encode(&attr.Value)
	}
}
