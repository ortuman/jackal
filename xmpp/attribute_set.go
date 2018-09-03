/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

import (
	"encoding/gob"
)

// Attribute represents an XML node attribute (label=value).
type Attribute struct {
	Label string
	Value string
}

// AttributeSet interface represents a read-only set of XML attributes.
type AttributeSet interface {
	Get(string) string
	Count() int
}

type attributeSet []Attribute

func (as attributeSet) Get(label string) string {
	for _, attr := range as {
		if attr.Label == label {
			return attr.Value
		}
	}
	return ""
}

func (as attributeSet) Count() int {
	return len(as)
}

func (as *attributeSet) setAttribute(label, value string) {
	for i := 0; i < len(*as); i++ {
		if (*as)[i].Label == label {
			(*as)[i].Value = value
			return
		}
	}
	*as = append(*as, Attribute{label, value})
}

func (as *attributeSet) removeAttribute(label string) {
	for i := 0; i < len(*as); i++ {
		if (*as)[i].Label == label {
			*as = append((*as)[:i], (*as)[i+1:]...)
			return
		}
	}
}

func (as *attributeSet) copyFrom(from attributeSet) {
	*as = make([]Attribute, from.Count())
	copy(*as, from)
}

func (as *attributeSet) fromGob(dec *gob.Decoder) {
	var c int
	dec.Decode(&c)
	for i := 0; i < c; i++ {
		var attr Attribute
		dec.Decode(&attr.Label)
		dec.Decode(&attr.Value)
		*as = append(*as, attr)
	}
}

func (as attributeSet) toGob(enc *gob.Encoder) {
	enc.Encode(len(as))
	for _, attr := range as {
		enc.Encode(&attr.Label)
		enc.Encode(&attr.Value)
	}
}
